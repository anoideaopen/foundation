package grpc

import (
	"fmt"
	"strings"

	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/reflect/protoregistry"
)

// FindServiceDescriptor finds the service descriptor by the given service name.
func FindServiceDescriptor(serviceName string) protoreflect.ServiceDescriptor {
	var sd protoreflect.ServiceDescriptor

	protoregistry.GlobalFiles.RangeFiles(func(fd protoreflect.FileDescriptor) bool {
		for i := range fd.Services().Len() {
			sd = fd.Services().Get(i)
			if sd.FullName() == protoreflect.FullName(serviceName) {
				return false
			}
		}
		return true
	})
	if sd == nil || sd.FullName() != protoreflect.FullName(serviceName) {
		return nil
	}

	return sd
}

// FullNameToURL transforms a method name from "package.Service.Method" to "/package.Service/Method"
func FullNameToURL(fullMethodName string) string {
	parts := strings.Split(fullMethodName, ".")
	if len(parts) < 2 {
		return ""
	}

	var (
		methodName  = parts[len(parts)-1]
		serviceName = parts[len(parts)-2]
		packageName = strings.Join(parts[:len(parts)-2], ".")
	)

	return fmt.Sprintf("/%s.%s/%s", packageName, serviceName, methodName)
}

// URLToServiceAndMethod extracts the service name and method name from a URL.
func URLToServiceAndMethod(url string) (string, string) {
	if len(url) == 0 || url[0] != '/' {
		return "", ""
	}

	// Split the trimmed URL by '/'
	parts := strings.Split(url[1:], "/")
	if len(parts) != 2 {
		return "", ""
	}

	var (
		serviceName = parts[0]
		methodName  = parts[1]
	)

	return serviceName, methodName
}
