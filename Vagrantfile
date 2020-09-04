Vagrant.configure("2") do |config|
  config.vm.box = "centos/7"

  # Broker
  config.vm.network "forwarded_port", guest: 8000, host: 8000
  
  # Install git
  config.vm.provision "shell", inline: <<-SHELL
    yum -y install git
  SHELL
  
  # Install golang and setup its environment
  config.vm.provision "shell", privileged: false, inline: <<-SHELL
    curl -O https://golang.org/dl/go1.14.8.linux-amd64.tar.gz
    mkdir -p go/src go/bin go/pkg
    echo "
    export GOPATH=$HOME/go
    export GOROOT=/usr/local/go
    export PATH=$PATH:/usr/local/go/bin:$HOME/go/bin
    " >> .bashrc
  SHELL

  config.vm.provision "shell", inline: <<-SHELL
    tar -C /usr/local -xzf go1.14.8.linux-amd64.tar.gz
  SHELL
end