test:
	go test ./...
test-Login:
	go test -v -run Test_AdminAuthenticate ./... 
test-getCustomerByMobile:
	go test -v -run Test_GetCustomerByMobile ./... 

.PHONY: test test-login test-getCustomerByMobile