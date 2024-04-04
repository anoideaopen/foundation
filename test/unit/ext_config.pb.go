// Code generated by protoc-gen-go. DO NOT EDIT.
// versions:
// 	protoc-gen-go v1.31.0
// 	protoc        v3.21.12
// source: ext_config.proto

package unit

import (
	reflect "reflect"
	sync "sync"

	proto "github.com/anoideaopen/foundation/proto"
	_ "github.com/envoyproxy/protoc-gen-validate/validate"
	protoreflect "google.golang.org/protobuf/reflect/protoreflect"
	protoimpl "google.golang.org/protobuf/runtime/protoimpl"
)

const (
	// Verify that this generated code is sufficiently up-to-date.
	_ = protoimpl.EnforceVersion(20 - protoimpl.MinVersion)
	// Verify that runtime/protoimpl is sufficiently up-to-date.
	_ = protoimpl.EnforceVersion(protoimpl.MaxVersion - 20)
)

type ExtConfig struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	Asset  string        `protobuf:"bytes,1,opt,name=asset,proto3" json:"asset,omitempty"`
	Amount string        `protobuf:"bytes,2,opt,name=amount,proto3" json:"amount,omitempty"`
	Issuer *proto.Wallet `protobuf:"bytes,3,opt,name=issuer,proto3" json:"issuer,omitempty"`
}

func (x *ExtConfig) Reset() {
	*x = ExtConfig{}
	if protoimpl.UnsafeEnabled {
		mi := &file_ext_config_proto_msgTypes[0]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *ExtConfig) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*ExtConfig) ProtoMessage() {}

func (x *ExtConfig) ProtoReflect() protoreflect.Message {
	mi := &file_ext_config_proto_msgTypes[0]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use ExtConfig.ProtoReflect.Descriptor instead.
func (*ExtConfig) Descriptor() ([]byte, []int) {
	return file_ext_config_proto_rawDescGZIP(), []int{0}
}

func (x *ExtConfig) GetAsset() string {
	if x != nil {
		return x.Asset
	}
	return ""
}

func (x *ExtConfig) GetAmount() string {
	if x != nil {
		return x.Amount
	}
	return ""
}

func (x *ExtConfig) GetIssuer() *proto.Wallet {
	if x != nil {
		return x.Issuer
	}
	return nil
}

var File_ext_config_proto protoreflect.FileDescriptor

var file_ext_config_proto_rawDesc = []byte{
	0x0a, 0x10, 0x65, 0x78, 0x74, 0x5f, 0x63, 0x6f, 0x6e, 0x66, 0x69, 0x67, 0x2e, 0x70, 0x72, 0x6f,
	0x74, 0x6f, 0x12, 0x04, 0x75, 0x6e, 0x69, 0x74, 0x1a, 0x0c, 0x63, 0x6f, 0x6e, 0x66, 0x69, 0x67,
	0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x1a, 0x17, 0x76, 0x61, 0x6c, 0x69, 0x64, 0x61, 0x74, 0x65,
	0x2f, 0x76, 0x61, 0x6c, 0x69, 0x64, 0x61, 0x74, 0x65, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x22,
	0x7c, 0x0a, 0x09, 0x45, 0x78, 0x74, 0x43, 0x6f, 0x6e, 0x66, 0x69, 0x67, 0x12, 0x1d, 0x0a, 0x05,
	0x61, 0x73, 0x73, 0x65, 0x74, 0x18, 0x01, 0x20, 0x01, 0x28, 0x09, 0x42, 0x07, 0xfa, 0x42, 0x04,
	0x72, 0x02, 0x10, 0x03, 0x52, 0x05, 0x61, 0x73, 0x73, 0x65, 0x74, 0x12, 0x1f, 0x0a, 0x06, 0x61,
	0x6d, 0x6f, 0x75, 0x6e, 0x74, 0x18, 0x02, 0x20, 0x01, 0x28, 0x09, 0x42, 0x07, 0xfa, 0x42, 0x04,
	0x72, 0x02, 0x10, 0x01, 0x52, 0x06, 0x61, 0x6d, 0x6f, 0x75, 0x6e, 0x74, 0x12, 0x2f, 0x0a, 0x06,
	0x69, 0x73, 0x73, 0x75, 0x65, 0x72, 0x18, 0x03, 0x20, 0x01, 0x28, 0x0b, 0x32, 0x0d, 0x2e, 0x70,
	0x72, 0x6f, 0x74, 0x6f, 0x2e, 0x57, 0x61, 0x6c, 0x6c, 0x65, 0x74, 0x42, 0x08, 0xfa, 0x42, 0x05,
	0x8a, 0x01, 0x02, 0x10, 0x01, 0x52, 0x06, 0x69, 0x73, 0x73, 0x75, 0x65, 0x72, 0x42, 0x09, 0x5a,
	0x07, 0x2e, 0x2f, 0x3b, 0x75, 0x6e, 0x69, 0x74, 0x62, 0x06, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x33,
}

var (
	file_ext_config_proto_rawDescOnce sync.Once
	file_ext_config_proto_rawDescData = file_ext_config_proto_rawDesc
)

func file_ext_config_proto_rawDescGZIP() []byte {
	file_ext_config_proto_rawDescOnce.Do(func() {
		file_ext_config_proto_rawDescData = protoimpl.X.CompressGZIP(file_ext_config_proto_rawDescData)
	})
	return file_ext_config_proto_rawDescData
}

var file_ext_config_proto_msgTypes = make([]protoimpl.MessageInfo, 1)
var file_ext_config_proto_goTypes = []interface{}{
	(*ExtConfig)(nil),    // 0: unit.ExtConfig
	(*proto.Wallet)(nil), // 1: proto.Wallet
}
var file_ext_config_proto_depIdxs = []int32{
	1, // 0: unit.ExtConfig.issuer:type_name -> proto.Wallet
	1, // [1:1] is the sub-list for method output_type
	1, // [1:1] is the sub-list for method input_type
	1, // [1:1] is the sub-list for extension type_name
	1, // [1:1] is the sub-list for extension extendee
	0, // [0:1] is the sub-list for field type_name
}

func init() { file_ext_config_proto_init() }
func file_ext_config_proto_init() {
	if File_ext_config_proto != nil {
		return
	}
	if !protoimpl.UnsafeEnabled {
		file_ext_config_proto_msgTypes[0].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*ExtConfig); i {
			case 0:
				return &v.state
			case 1:
				return &v.sizeCache
			case 2:
				return &v.unknownFields
			default:
				return nil
			}
		}
	}
	type x struct{}
	out := protoimpl.TypeBuilder{
		File: protoimpl.DescBuilder{
			GoPackagePath: reflect.TypeOf(x{}).PkgPath(),
			RawDescriptor: file_ext_config_proto_rawDesc,
			NumEnums:      0,
			NumMessages:   1,
			NumExtensions: 0,
			NumServices:   0,
		},
		GoTypes:           file_ext_config_proto_goTypes,
		DependencyIndexes: file_ext_config_proto_depIdxs,
		MessageInfos:      file_ext_config_proto_msgTypes,
	}.Build()
	File_ext_config_proto = out.File
	file_ext_config_proto_rawDesc = nil
	file_ext_config_proto_goTypes = nil
	file_ext_config_proto_depIdxs = nil
}
