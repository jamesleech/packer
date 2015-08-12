if((Test-Path .\build) -ne $true) {
    mkdir build
}

# clean old build
rm .\build\*

# build
go build
go build .\plugin\builder-hyperv-iso 
go build .\plugin\builder-docker
go build .\plugin\builder-googlecompute
go build .\plugin\builder-parallels-iso
go build .\plugin\builder-qemu
go build .\plugin\builder-virtualbox-iso
go build .\plugin\builder-vmware-iso
go build .\plugin\builder-null
go build .\plugin\post-processor-compress
go build .\plugin\post-processor-docker-import
go build .\plugin\post-processor-docker-push
go build .\plugin\post-processor-docker-save
go build .\plugin\post-processor-docker-tag
go build .\plugin\post-processor-vagrant
go build .\plugin\post-processor-vsphere
go build .\plugin\provisioner-chef-client
go build .\plugin\provisioner-chef-solo
go build .\plugin\provisioner-file
go build .\plugin\provisioner-shell

$exes = Get-ChildItem -Recurse .\*.exe
$exes | ForEach-Object -Process {copy $_ .\build}
$exes | ForEach-Object -Process {rm $_ }

dir .\build