module github.com/dan88c/wallop/tests/go_tests

go 1.22

require github.com/dan88c/wallop/core-operator v0.0.0

require gopkg.in/yaml.v3 v3.0.1 // indirect

replace github.com/dan88c/wallop/core-operator => ../../core-operator
