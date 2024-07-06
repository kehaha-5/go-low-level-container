./clear.sh > /dev/null 2>&1 
rm -rf /usr/mydocker/container/ /usr/mydocker/info/ /usr/mydocker/network/
/workspaces/go-low-level-simple-runc/build/simple-docker network create -d bridge --subnet 192.168.0.1/24 web
/workspaces/go-low-level-simple-runc/build/simple-docker run -d --name ng -v /workspaces/go-low-level-simple-runc/simple_docker/nginx-v/conf:/etc/nginx/conf.d/ -v /workspaces/go-low-level-simple-runc/simple_docker/nginx-v/html/:/usr/share/nginx/html -p 40624:80 -net web mynginx ./docker-entrypoint.sh nginx
/workspaces/go-low-level-simple-runc/build/simple-docker run  -d --name myphp -v /workspaces/go-low-level-simple-runc/simple_docker/php_v:/var/www/html/ -net web  myphpfpm sh ./start-php-fpm.sh
