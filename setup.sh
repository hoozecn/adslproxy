#!/bin/bash

umask 066
mkdir -p ~/.ssh

cat > ~/.ssh/authorized_keys <<EOF
ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAABAQC4LsRjj2jm+Mvjl0dqV9l9JyRzns7RATaZq9QjS1NcOMRSGtm+kaX0jU+7Rw/IjPhyJKzgZDAxH4prabNrnk+5e/Nv2181hLANdenyiNEtu+K1LX4bZKUX6ayERlPM0PlDCM11HiQBhefFBTK7yKyxFzRddhXHHck76En77qr3HQqwswf5V/yu3dIjVuN5tPF/HJtRmOpjgo6pdbecAzGgOBh/dtwNlwQilhJ1axNRHLzT6d4/4dP0o2l0KzBKeMrmueXLySKZ8RyYMjeR1kL7eHJSgGFysRqx+7tWTnHE8OFcKa0qC8fRb0XEmUeAk7fbFmJGH8EQeEBgGns2BWtB hoozecn@hoozecn-T400
EOF

yum install -y squid tmux

cat > /etc/squid/squid.conf <<EOF
http_access allow all

# Squid normally listens to port 3128
http_port 3128
EOF

service squid restart
