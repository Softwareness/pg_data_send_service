rm -rf main
rm -rf main.zip
GOOS=linux go build -o main main.go
zip function.zip main