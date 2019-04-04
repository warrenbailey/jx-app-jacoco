OS := $(shell uname | tr '[:upper:]' '[:lower:]')

tag:
ifeq ($(OS),darwin)
	sed -i "" -e "s/version:.*/version: $(VERSION)/" Chart.yaml
	sed -i "" -e "s/tag: .*/tag: $(VERSION)/" values.yaml
else ifeq ($(OS),linux)
	sed -i -e "s/version:.*/version: $(VERSION)/" Chart.yaml
	sed -i -e "s/tag: .*/tag: $(VERSION)/" values.yaml
else
	echo "platform $(OS) not supported to tag with"
	exit -1
endif