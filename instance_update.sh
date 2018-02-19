#!/bin/bash
source <( curl "http://169.254.169.254/latest/user-data" 2>/dev/null )
 
if [ "${GO_GIT_HASH}" != "" ]; then
  pushd $PROJECT_DIR >/dev/null 2>/dev/null
  su go -c "vendor/github.com/rem7/servercontrol/git_update_to_hash.sh ${GO_PROJECT} ${GO_GIT_HASH}" >> /tmp/instance-update.log 2>&1
  popd >/dev/null 2>/dev/null
  ln -s /etc/sv/$GO_PROJECT /etc/service/$GO_PROJECT
fi


