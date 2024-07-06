/workspaces/go-low-level-simple-runc/build/simple-docker network remove web 
/workspaces/go-low-level-simple-runc/build/simple-docker rm -f ng
/workspaces/go-low-level-simple-runc/build/simple-docker rm -f myphp
ip netns delete ng
ip netns delete myphp