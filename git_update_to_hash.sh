#!/bin/bash
 
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
 
echo "Pulling latest."
git pull
if [ "$?" -ne 0 ]; then
    echo " - Pull failed."
    exit 1
fi
echo "Checking out git_hash $git_hash."
git checkout -q $git_hash
if [ "$?" -ne 0 ]; then
    echo " - Checkout failed."
    if [ "$2" != "" ]; then
        echo "Reverting to git_hash revert_hash."
        git checkout -q $revert_hash
        if [ "$?" -ne 0 ]; then
            echo " - Revert failed!"
            exit 2
        fi
        echo " - Revert Success."
    fi
    exit 1
else

    # forget about deps for now and checkin vendor
    # $GOPATH/bin/glide install
    # if [ "$?" -ne 0 ]; then 
    #     echo " - Updating deps failed. Is dep installed?"
    #     exit 3
    # fi

    # # don't allow lock after install
    # # this should allow glide.lock if the repo has it commited
    # rm glide.lock

    # forget about building. run main.go
    # echo ". building "
    # /usr/local/go/bin/go build -v -ldflags "-X main.gitHash=`git rev-parse HEAD`" -o /tmp/$build_name
    # if [ "$?" -ne 0 ]; then 
    #     echo " - Compiling failed."
    #     exit 4
    # fi
fi

exit 0

