// Code generated by protoc-gen-go. DO NOT EDIT.
// versions:
// 	protoc-gen-go v1.30.0
// 	protoc        v4.22.5
// source: report.proto

package proto

import (
	protoreflect "google.golang.org/protobuf/reflect/protoreflect"
	protoimpl "google.golang.org/protobuf/runtime/protoimpl"
	reflect "reflect"
	sync "sync"
)

const (
	// Verify that this generated code is sufficiently up-to-date.
	_ = protoimpl.EnforceVersion(20 - protoimpl.MinVersion)
	// Verify that runtime/protoimpl is sufficiently up-to-date.
	_ = protoimpl.EnforceVersion(protoimpl.MaxVersion - 20)
)

type Report struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	FeePublicKey     []byte  `protobuf:"bytes,1,opt,name=fee_public_key,json=feePublicKey,proto3" json:"fee_public_key,omitempty"`
	ChecksumOrderer  uint64  `protobuf:"varint,2,opt,name=checksum_orderer,json=checksumOrderer,proto3" json:"checksum_orderer,omitempty"`
	ChecksumEndorser uint64  `protobuf:"varint,3,opt,name=checksum_endorser,json=checksumEndorser,proto3" json:"checksum_endorser,omitempty"`
	Stats            []*Stat `protobuf:"bytes,4,rep,name=stats,proto3" json:"stats,omitempty"`
}

func (x *Report) Reset() {
	*x = Report{}
	if protoimpl.UnsafeEnabled {
		mi := &file_report_proto_msgTypes[0]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *Report) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*Report) ProtoMessage() {}

func (x *Report) ProtoReflect() protoreflect.Message {
	mi := &file_report_proto_msgTypes[0]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use Report.ProtoReflect.Descriptor instead.
func (*Report) Descriptor() ([]byte, []int) {
	return file_report_proto_rawDescGZIP(), []int{0}
}

func (x *Report) GetFeePublicKey() []byte {
	if x != nil {
		return x.FeePublicKey
	}
	return nil
}

func (x *Report) GetChecksumOrderer() uint64 {
	if x != nil {
		return x.ChecksumOrderer
	}
	return 0
}

func (x *Report) GetChecksumEndorser() uint64 {
	if x != nil {
		return x.ChecksumEndorser
	}
	return 0
}

func (x *Report) GetStats() []*Stat {
	if x != nil {
		return x.Stats
	}
	return nil
}

type Stat struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	CertOwner      string `protobuf:"bytes,1,opt,name=cert_owner,json=certOwner,proto3" json:"cert_owner,omitempty"`
	PointsEndorser uint64 `protobuf:"varint,2,opt,name=points_endorser,json=pointsEndorser,proto3" json:"points_endorser,omitempty"`
	PointsOrderer  uint64 `protobuf:"varint,3,opt,name=points_orderer,json=pointsOrderer,proto3" json:"points_orderer,omitempty"`
}

func (x *Stat) Reset() {
	*x = Stat{}
	if protoimpl.UnsafeEnabled {
		mi := &file_report_proto_msgTypes[1]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *Stat) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*Stat) ProtoMessage() {}

func (x *Stat) ProtoReflect() protoreflect.Message {
	mi := &file_report_proto_msgTypes[1]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use Stat.ProtoReflect.Descriptor instead.
func (*Stat) Descriptor() ([]byte, []int) {
	return file_report_proto_rawDescGZIP(), []int{1}
}

func (x *Stat) GetCertOwner() string {
	if x != nil {
		return x.CertOwner
	}
	return ""
}

func (x *Stat) GetPointsEndorser() uint64 {
	if x != nil {
		return x.PointsEndorser
	}
	return 0
}

func (x *Stat) GetPointsOrderer() uint64 {
	if x != nil {
		return x.PointsOrderer
	}
	return 0
}

type HeadInfo struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	Heads []*Head `protobuf:"bytes,1,rep,name=heads,proto3" json:"heads,omitempty"`
}

func (x *HeadInfo) Reset() {
	*x = HeadInfo{}
	if protoimpl.UnsafeEnabled {
		mi := &file_report_proto_msgTypes[2]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *HeadInfo) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*HeadInfo) ProtoMessage() {}

func (x *HeadInfo) ProtoReflect() protoreflect.Message {
	mi := &file_report_proto_msgTypes[2]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use HeadInfo.ProtoReflect.Descriptor instead.
func (*HeadInfo) Descriptor() ([]byte, []int) {
	return file_report_proto_rawDescGZIP(), []int{2}
}

func (x *HeadInfo) GetHeads() []*Head {
	if x != nil {
		return x.Heads
	}
	return nil
}

type Head struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	Token    string `protobuf:"bytes,1,opt,name=token,proto3" json:"token,omitempty"`
	BlockNum uint64 `protobuf:"varint,2,opt,name=block_num,json=blockNum,proto3" json:"block_num,omitempty"`
}

func (x *Head) Reset() {
	*x = Head{}
	if protoimpl.UnsafeEnabled {
		mi := &file_report_proto_msgTypes[3]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *Head) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*Head) ProtoMessage() {}

func (x *Head) ProtoReflect() protoreflect.Message {
	mi := &file_report_proto_msgTypes[3]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use Head.ProtoReflect.Descriptor instead.
func (*Head) Descriptor() ([]byte, []int) {
	return file_report_proto_rawDescGZIP(), []int{3}
}

func (x *Head) GetToken() string {
	if x != nil {
		return x.Token
	}
	return ""
}

func (x *Head) GetBlockNum() uint64 {
	if x != nil {
		return x.BlockNum
	}
	return 0
}

var File_report_proto protoreflect.FileDescriptor

var file_report_proto_rawDesc = []byte{
	0x0a, 0x0c, 0x72, 0x65, 0x70, 0x6f, 0x72, 0x74, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x12, 0x05,
	0x70, 0x72, 0x6f, 0x74, 0x6f, 0x22, 0xa9, 0x01, 0x0a, 0x06, 0x52, 0x65, 0x70, 0x6f, 0x72, 0x74,
	0x12, 0x24, 0x0a, 0x0e, 0x66, 0x65, 0x65, 0x5f, 0x70, 0x75, 0x62, 0x6c, 0x69, 0x63, 0x5f, 0x6b,
	0x65, 0x79, 0x18, 0x01, 0x20, 0x01, 0x28, 0x0c, 0x52, 0x0c, 0x66, 0x65, 0x65, 0x50, 0x75, 0x62,
	0x6c, 0x69, 0x63, 0x4b, 0x65, 0x79, 0x12, 0x29, 0x0a, 0x10, 0x63, 0x68, 0x65, 0x63, 0x6b, 0x73,
	0x75, 0x6d, 0x5f, 0x6f, 0x72, 0x64, 0x65, 0x72, 0x65, 0x72, 0x18, 0x02, 0x20, 0x01, 0x28, 0x04,
	0x52, 0x0f, 0x63, 0x68, 0x65, 0x63, 0x6b, 0x73, 0x75, 0x6d, 0x4f, 0x72, 0x64, 0x65, 0x72, 0x65,
	0x72, 0x12, 0x2b, 0x0a, 0x11, 0x63, 0x68, 0x65, 0x63, 0x6b, 0x73, 0x75, 0x6d, 0x5f, 0x65, 0x6e,
	0x64, 0x6f, 0x72, 0x73, 0x65, 0x72, 0x18, 0x03, 0x20, 0x01, 0x28, 0x04, 0x52, 0x10, 0x63, 0x68,
	0x65, 0x63, 0x6b, 0x73, 0x75, 0x6d, 0x45, 0x6e, 0x64, 0x6f, 0x72, 0x73, 0x65, 0x72, 0x12, 0x21,
	0x0a, 0x05, 0x73, 0x74, 0x61, 0x74, 0x73, 0x18, 0x04, 0x20, 0x03, 0x28, 0x0b, 0x32, 0x0b, 0x2e,
	0x70, 0x72, 0x6f, 0x74, 0x6f, 0x2e, 0x53, 0x74, 0x61, 0x74, 0x52, 0x05, 0x73, 0x74, 0x61, 0x74,
	0x73, 0x22, 0x75, 0x0a, 0x04, 0x53, 0x74, 0x61, 0x74, 0x12, 0x1d, 0x0a, 0x0a, 0x63, 0x65, 0x72,
	0x74, 0x5f, 0x6f, 0x77, 0x6e, 0x65, 0x72, 0x18, 0x01, 0x20, 0x01, 0x28, 0x09, 0x52, 0x09, 0x63,
	0x65, 0x72, 0x74, 0x4f, 0x77, 0x6e, 0x65, 0x72, 0x12, 0x27, 0x0a, 0x0f, 0x70, 0x6f, 0x69, 0x6e,
	0x74, 0x73, 0x5f, 0x65, 0x6e, 0x64, 0x6f, 0x72, 0x73, 0x65, 0x72, 0x18, 0x02, 0x20, 0x01, 0x28,
	0x04, 0x52, 0x0e, 0x70, 0x6f, 0x69, 0x6e, 0x74, 0x73, 0x45, 0x6e, 0x64, 0x6f, 0x72, 0x73, 0x65,
	0x72, 0x12, 0x25, 0x0a, 0x0e, 0x70, 0x6f, 0x69, 0x6e, 0x74, 0x73, 0x5f, 0x6f, 0x72, 0x64, 0x65,
	0x72, 0x65, 0x72, 0x18, 0x03, 0x20, 0x01, 0x28, 0x04, 0x52, 0x0d, 0x70, 0x6f, 0x69, 0x6e, 0x74,
	0x73, 0x4f, 0x72, 0x64, 0x65, 0x72, 0x65, 0x72, 0x22, 0x2d, 0x0a, 0x08, 0x48, 0x65, 0x61, 0x64,
	0x49, 0x6e, 0x66, 0x6f, 0x12, 0x21, 0x0a, 0x05, 0x68, 0x65, 0x61, 0x64, 0x73, 0x18, 0x01, 0x20,
	0x03, 0x28, 0x0b, 0x32, 0x0b, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x2e, 0x48, 0x65, 0x61, 0x64,
	0x52, 0x05, 0x68, 0x65, 0x61, 0x64, 0x73, 0x22, 0x39, 0x0a, 0x04, 0x48, 0x65, 0x61, 0x64, 0x12,
	0x14, 0x0a, 0x05, 0x74, 0x6f, 0x6b, 0x65, 0x6e, 0x18, 0x01, 0x20, 0x01, 0x28, 0x09, 0x52, 0x05,
	0x74, 0x6f, 0x6b, 0x65, 0x6e, 0x12, 0x1b, 0x0a, 0x09, 0x62, 0x6c, 0x6f, 0x63, 0x6b, 0x5f, 0x6e,
	0x75, 0x6d, 0x18, 0x02, 0x20, 0x01, 0x28, 0x04, 0x52, 0x08, 0x62, 0x6c, 0x6f, 0x63, 0x6b, 0x4e,
	0x75, 0x6d, 0x42, 0x0a, 0x5a, 0x08, 0x2e, 0x2f, 0x3b, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x62, 0x06,
	0x70, 0x72, 0x6f, 0x74, 0x6f, 0x33,
}

var (
	file_report_proto_rawDescOnce sync.Once
	file_report_proto_rawDescData = file_report_proto_rawDesc
)

func file_report_proto_rawDescGZIP() []byte {
	file_report_proto_rawDescOnce.Do(func() {
		file_report_proto_rawDescData = protoimpl.X.CompressGZIP(file_report_proto_rawDescData)
	})
	return file_report_proto_rawDescData
}

var file_report_proto_msgTypes = make([]protoimpl.MessageInfo, 4)
var file_report_proto_goTypes = []interface{}{
	(*Report)(nil),   // 0: proto.Report
	(*Stat)(nil),     // 1: proto.Stat
	(*HeadInfo)(nil), // 2: proto.HeadInfo
	(*Head)(nil),     // 3: proto.Head
}
var file_report_proto_depIdxs = []int32{
	1, // 0: proto.Report.stats:type_name -> proto.Stat
	3, // 1: proto.HeadInfo.heads:type_name -> proto.Head
	2, // [2:2] is the sub-list for method output_type
	2, // [2:2] is the sub-list for method input_type
	2, // [2:2] is the sub-list for extension type_name
	2, // [2:2] is the sub-list for extension extendee
	0, // [0:2] is the sub-list for field type_name
}

func init() { file_report_proto_init() }
func file_report_proto_init() {
	if File_report_proto != nil {
		return
	}
	if !protoimpl.UnsafeEnabled {
		file_report_proto_msgTypes[0].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*Report); i {
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
		file_report_proto_msgTypes[1].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*Stat); i {
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
		file_report_proto_msgTypes[2].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*HeadInfo); i {
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
		file_report_proto_msgTypes[3].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*Head); i {
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
			RawDescriptor: file_report_proto_rawDesc,
			NumEnums:      0,
			NumMessages:   4,
			NumExtensions: 0,
			NumServices:   0,
		},
		GoTypes:           file_report_proto_goTypes,
		DependencyIndexes: file_report_proto_depIdxs,
		MessageInfos:      file_report_proto_msgTypes,
	}.Build()
	File_report_proto = out.File
	file_report_proto_rawDesc = nil
	file_report_proto_goTypes = nil
	file_report_proto_depIdxs = nil
}
