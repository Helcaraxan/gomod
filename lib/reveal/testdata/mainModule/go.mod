module test/module

go 1.12

require (
    module/foo v1.0.0
    module/bar v1.0.0
)

replace module/foo => module/foo-bis v1.0.0
