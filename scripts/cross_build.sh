GOOS=windows GOARCH=amd64 go build main.go
zip release.zip main.exe ./static/css/* ./static/js/* ./static/*