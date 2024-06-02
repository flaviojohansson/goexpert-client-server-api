module github.com/flaviojohansson/goexpert-client-server-api/server

go 1.22.0

require (
	github.com/flaviojohansson/goexpert-client-server-api/common v0.0.0-00010101000000-000000000000
	github.com/valyala/fastjson v1.6.4
)

replace github.com/flaviojohansson/goexpert-client-server-api/common => ../common
