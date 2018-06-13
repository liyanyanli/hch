#!/bin/bash
bashpath=$(cd `dirname $0`; pwd)

function color_echo() {
    if [ $1 == "green" ]; then
        echo -e "\033[32;40m$2\033[0m"
    elif [ $1 == "red" ]; then
        echo -e "\033[31;40m$2\033[0m"
    fi
}
function os_version() {
    local OS_V=$(cat /etc/issue |awk 'NR==1{print $1}')
    if [ $OS_V == "\S" -o $OS_V == "CentOS" ]; then
        echo "CentOS"
    elif [ $OS_V == "Ubuntu" ]; then
        echo "Ubuntu"
    fi
}
function check_ssh_auth() {
    if $(grep "Permission denied" $EXP_TMP_FILE >/dev/null); then
        color_echo red "[Host $IP] SSH authentication failure! Login password error."
        exit 1
    elif $(ssh $INFO 'echo yes >/dev/null'); then
        color_echo green "[Host $IP] SSH authentication successfully."
    fi
    #rm $EXP_TMP_FILE >/dev/null
}
function check_hostname() {
    ssh -t $INFO hostname > $EXP_TMP_FILE
    if $(grep "$IP" $EXP_TMP_FILE >/dev/null); then
        color_echo green "[Host $IP] Hostname is correct."
    else
       color_echo red "[Host $IP] Hostname is not IP !"
       exit 1
    fi
    rm $EXP_TMP_FILE >/dev/null
}
function check_pkg() {
    local PKG_NAME=$1
    if [ $(os_version) == "CentOS" ]; then
        if ! $(rpm -ql $PKG_NAME >/dev/null 2>&1); then
            echo no
        else
            echo yes
        fi
    elif [ $(os_version) == "Ubuntu" ]; then
        if ! $(dpkg -l $PKG_NAME >/dev/null 2>&1); then
            echo no
        else
            echo yes
        fi
    fi
}
function install_pkg() {
    local PKG_NAME=$1
    if [ $(os_version) == "CentOS" ]; then
        if [ $(check_pkg $PKG_NAME) == "no" ]; then
            yum install $PKG_NAME -y
            if [ $(check_pkg $PKG_NAME) == "no" ]; then
                color_echo green "The $PKG_NAME installation failure! Try to install again."
                yum makecache
                yum install $PKG_NAME -y
                [ $(check_pkg $PKG_NAME) == "no" ] && color_echo red "The $PKG_NAME installation failure!" && exit 1
            fi
        fi
    elif [ $(os_version) == "Ubuntu" ]; then
        if [ $(check_pkg $PKG_NAME) == "no" ]; then
            apt-get install $PKG_NAME -y
            if [ $(check_pkg $PKG_NAME) == "no" ]; then
                color_echo green "$PKG_NAME installation failure! Try to install again."
                apt-get autoremove && apt-get update
                apt-get install $PKG_NAME --force-yes -y
                [ $(check_pkg $PKG_NAME) == "no" ] && color_echo red "The $PKG_NAME installation failure!" && exit 1
            fi
        fi
    fi
}
function generate_keypair() {
    if [ ! -e /root/.ssh/id_rsa.pub ]; then
        color_echo green "The public/private rsa key pair not exist, start Generating..."
        expect -c "
            spawn ssh-keygen
            expect {
                \"ssh/id_rsa):\" {send \"\r\";exp_continue}
                \"passphrase):\" {send \"\r\";exp_continue}
                \"again:\" {send \"\r\";exp_continue}
            }
        " >/dev/null 2>&1
        if [ -e /root/.ssh/id_rsa.pub ]; then
            color_echo green "Generating public/private rsa key pair successfully."
        else
            color_echo red "Generating public/private rsa key pair failure!"
            exit 1
        fi
    fi
}
 
EXP_TMP_FILE=/tmp/expect_ssh.tmp

#if [ $(check_pkg expect) == "no" ]; then
#   rpm -ivh $bashpath/expect-centos7-2/tcl-8.5.13-8.el7.x86_64.rpm --nodeps --force
#   rpm -ivh $bashpath/expect-centos7-2/expect-5.45-14.el7_1.x86_64.rpm --nodeps --force
#fi
generate_keypair

mkdir -p /home/.ssh
cp -rf /root/.ssh/id_rsa.pub /home/.ssh

USER=$1
IP=$2
PASS=$3
INFO=$USER@$IP
expect -c "
        spawn ssh-copy-id -i /root/.ssh/id_rsa.pub $INFO
        expect {
            \"(yes/no)?\" {send \"yes\r\";exp_continue}
            \"password:\" {send \"$PASS\r\";exp_continue}
        }
" > $EXP_TMP_FILE  # if login failed, login error info append temp file
check_ssh_auth


