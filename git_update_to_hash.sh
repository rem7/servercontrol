#!/bin/bash
 
source <( curl "http://169.254.169.254/latest/user-data" 2>/dev/null )

if [ "$#" -lt 1 ]; then
    echo "Usage: $git_hash <app_name> <git_hash> [revert_hash]"
    exit 1
fi

PATH=$PATH:/usr/local/go/bin/

appname=$1
git_hash=$2
revert_hash=$3
build_name=$1-$2

 
echo "Discard any local changes"
git checkout -f 
# git clean -f
echo "Checkout master"
git checkout -q master
if [ "$?" -ne 0 ]; then
    echo " - Checkout master failed."
    exit 1
fi
 
echo "Pulling latest. -- forced"
git fetch origin master
git reset --hard FETCH_HEAD
git clean -df
if [ "$?" -ne 0 ]; then
    echo " - Pull failed."
    exit 1
fi
echo "Checking out git_hash $git_hash."
git checkout -q $git_hash
if [ "$?" -ne 0 ]; then
    echo " - Checkout failed."
    if [ "$2" != "" ]; then
        echo "Reverting to git_hash $revert_hash."
        git checkout -q $revert_hash
        if [ "$?" -ne 0 ]; then
            echo " - Revert failed!"
            exit 2
        fi
        echo " - Revert Success."
    fi
    exit 1
else

    export GOPATH=$GOPATH
    /var/go/src/go/bin/dep ensure
    
    echo ". building "
    /usr/local/go/bin/go build -v -ldflags "-X main.gitHash=`git rev-parse HEAD`" -o /tmp/$build_name
    if [ "$?" -ne 0 ]; then 
        echo " - Compiling failed."
        echo "Reverting to git_hash $revert_hash."
        git checkout -q $revert_hash
        if [ "$?" -ne 0 ]; then
            echo " - Revert failed!"
            exit 2
        fi
        echo " - Revert Success."
        exit 4
    fi
fi

exit 0

