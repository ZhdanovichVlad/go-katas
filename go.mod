module github.com/ZhdanovichVlad/go-katas

go 1.26.2

require (
	github.com/lib/pq v1.12.3
	golang.org/x/sync v0.20.0
)

require go.uber.org/goleak v1.3.0 // indirect

require (
	github.com/davecgh/go-spew v1.1.1 // indirect
	github.com/pmezard/go-difflib v1.0.0 // indirect
	github.com/stretchr/testify v1.10.0 // indirect
	gitlab.com/slon/shad-go v0.0.0-20250520111742-6718601587ec
	gopkg.in/yaml.v3 v3.0.1 // indirect
)

replace gitlab.com/slon/shad-go => github.com/slon/shad-go v0.0.0-20250520111742-6718601587ec
