# hc
helm chart   

## build and save Docker image
```
make pkg
```
## bucket helm chart package
```
helm package charts/hc/
```

```
1. NFS server
yum install nfs-utils
mkdir -p /home/share
echo '/home/share 172.16.10.0/16(rw,sync,no_all_squash,no_root_squash)' >> /etc/exports
systemctl start nfs-server
systemctl enable nsf-server

2. NFS client
mount -f nfs 172.16.10.123:/home/share /mnt/share
```

