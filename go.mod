module simple_port_forwarding

go 1.24.1

replace github.com/sdf1979/logger => ./pkg/rotatingLogger

replace github.com/sdf1979/appService => ./pkg/appService

replace github.com/sdf1979/config => ./pkg/config

require (
	github.com/sdf1979/appService v0.0.0-00010101000000-000000000000
	github.com/sdf1979/logger v0.0.0-00010101000000-000000000000
)

require (
	github.com/sdf1979/config v0.0.0-00010101000000-000000000000
	golang.org/x/sys v0.31.0
)
