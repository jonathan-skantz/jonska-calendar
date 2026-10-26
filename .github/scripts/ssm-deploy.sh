#!/usr/bin/env bash
# Deploys a release tag on the EC2 instance through AWS Systems Manager (SSM)
# Run Command, waits for it to finish, and prints the instance's output.
#
# Usage: ssm-deploy.sh TAG
#   INSTANCE_ID=i-0123... DEPLOY_USER=ubuntu DEPLOY_DIR=/home/ubuntu/calendar \
#     ssm-deploy.sh v1.3.0
#
# Needs AWS credentials allowed to run ssm:SendCommand on the instance (see
# docs/deployment.md). TAG must already be validated as vX.Y.Z: it is pasted
# into the command that runs on the server.
set -euo pipefail

tag=$1
: "${INSTANCE_ID:?}" "${DEPLOY_USER:?}" "${DEPLOY_DIR:?}"

# SSM runs commands as root. This one switches to the user who owns the clone,
# checks out the release, then runs deploy.sh from that checkout. The checkout
# happens here rather than in deploy.sh, so the script never changes while it
# runs. The command run on the server looks like:
#   sudo -iu ubuntu sh -c 'cd /home/ubuntu/calendar && git fetch ... && ./deploy.sh v1.3.0'
script="cd $DEPLOY_DIR && git fetch --tags --force && git checkout --detach $tag && ./deploy.sh $tag"
params=$(jq -nc --arg cmd "sudo -iu $DEPLOY_USER sh -c '$script'" '{commands: [$cmd]}')

command_id=$(aws ssm send-command \
  --instance-ids "$INSTANCE_ID" \
  --document-name AWS-RunShellScript \
  --comment "deploy $tag" \
  --parameters "$params" \
  --query Command.CommandId --output text)

invocation() {
  aws ssm get-command-invocation --command-id "$command_id" --instance-id "$INSTANCE_ID" "$@"
}

# Poll for up to 10 minutes. `aws ssm wait command-executed` gives up after
# 100 seconds, too soon for a first deploy that pulls every image. Right after
# send-command the invocation may not exist yet, which counts as pending.
for _ in $(seq 120); do
  sleep 5
  status=$(invocation --query Status --output text 2> /dev/null || echo Pending)
  case $status in
    Pending | InProgress | Delayed) ;;
    *) break ;;
  esac
done

result=$(invocation)
jq -r .StandardOutputContent <<< "$result"
jq -r .StandardErrorContent <<< "$result" >&2
status=$(jq -r .Status <<< "$result")
if [[ $status != Success ]]; then
  echo "::error::Deploy on the instance ended with status $status" >&2
  exit 1
fi
