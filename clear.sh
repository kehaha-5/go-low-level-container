./build/simple-docker network remove web 
./build/simple-docker rm -f ng
./build/simple-docker rm -f myphp
ip netns delete ng
ip netns delete myphp