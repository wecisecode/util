




CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go test -o ~/go/bin/linux_amd64/bfatest -c -args

CGO_ENABLED=0 go test -o bfatest -c -args; ./bfatest -test.run . parallel=2 loop=1; ls -lrt _test

