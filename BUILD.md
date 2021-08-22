# Build

## Get Dependencies without Building
### Go Module
```
go get -d github.com/jonoton/scout@v1.17.0
```
### Cloned
```
go get -d ./...
```

## Dependencies
### GoCV
#### Navigate to folder
```
cd $GOPATH/pkg/mod/gocv.io/x/gocv@v0.27.0
```
#### Choose One
##### Build
```
sudo make -j install
```
##### Build w/ CUDA Support
```
sudo make -j install_cuda
```

## Install Scout
### Go Module
```
go get github.com/jonoton/scout@v1.17.0
```
### Cloned
```
go install ./...
```

## Verify
```
ls -alh $GOPATH/bin
```
