#!/usr/bin/env bash

#example:sh build.sh /home/odl/Docker/develop/docker-harbor 192.168.1.1 smtp.qq.com 25 111111111@qq.com aaaaaaaa 11111111@qq.com aaaa

#cd $1/Deploy

#'DOCKER_OPTS="--insecure-registry '$2'"' >> /etc/default/docker
#
#service docker stop
#
#service docker start


#sed 's/hostname = reg.mydomain.com/hostname = '$2'/' harbor.cfg > harborR.cfg

#change email(email server)
#sed 's/email_server = smtp.mydomain.com/email_server = '$3'/' harborR.cfg > harbor.cfg

#change email(email port)
#sed 's/email_server_port = 25/email_server_port = '$4'/' harbor.cfg > harborR.cfg

#change email(email username)
#sed 's/email_username = sample_admin@mydomain.com/email_username = '$5'/' harborR.cfg > harbor.cfg

#change email(email password)
#sed 's/email_password = abc/email_password = '$6'/' harbor.cfg > harborR.cfg

#change email(email from)
#sed 's/email_from = admin <sample_admin@mydomain.com>/email_from = '$7'/' harborR.cfg > harbor.cfg

#change admin password
#sed 's/harbor_admin_password = Harbor12345/harbor_admin_password = '$8'/' harbor.cfg > harborR.cfg

#cat harborR.cfg > harbor.cfg

#rm -rf harborR.cfg

#harbor install

docker-compose down

./prepare

docker-compose up --build -d

#config ui ssh

uiUUID=`docker ps | grep deploy_ui | awk -F ' ' '{print $1}'`

docker exec -it $uiUUID rm -rf ~/.ssh/

docker exec -it $uiUUID /usr/bin/ssh-keygen -t rsa -f ~/.ssh/id_rsa -P ''

docker exec -it $uiUUID cat /root/.ssh/id_rsa.pub >> /root/.ssh/authorized_keys

#config mysql
#
#mysqlUUID=`docker ps | grep deploy_mysql | awk -F ' ' '{print $1}'`
#
#docker exec -it $mysqlUUID mv /etc/mysql/my.cnf /etc/mysql/my.cnf.old
#
#docker exec -it $mysqlUUID sed 's/sql_mode=NO_ENGINE_SUBSTITUTION,STRICT_TRANS_TABLES/sql_mode=NO_AUTO_CREATE_USER,NO_ENGINE_SUBSTITUTION/' /etc/mysql/my.cnf.old > /etc/mysql/my.cnf
#
#docker exec -it $mysqlUUID service mysql restart



