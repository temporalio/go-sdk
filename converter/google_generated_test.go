package converter

import (
	reflect "reflect"
	sync "sync"

	proto "github.com/golang/protobuf/proto"
	protoreflect "google.golang.org/protobuf/reflect/protoreflect"
	protoimpl "google.golang.org/protobuf/runtime/protoimpl"
)

const (
	// Verify that this generated code is sufficiently up-to-date.
	_ = protoimpl.EnforceVersion(20 - protoimpl.MinVersion)
	// Verify that runtime/protoimpl is sufficiently up-to-date.
	_ = protoimpl.EnforceVersion(protoimpl.MaxVersion - 20)
)

// This is a compile-time assertion that a sufficiently up-to-date version
// of the legacy proto package is being used.
const _ = proto.ProtoPackageIsVersion4

type GoogleGenerated struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	Name     string  `protobuf:"bytes,1,opt,name=name,proto3" json:"name,omitempty"`
	BirthDay int64   `protobuf:"varint,2,opt,name=birthDay,proto3" json:"birthDay,omitempty"`
	Phone    string  `protobuf:"bytes,3,opt,name=phone,proto3" json:"phone,omitempty"`
	Siblings int32   `protobuf:"varint,4,opt,name=siblings,proto3" json:"siblings,omitempty"`
	Spouse   bool    `protobuf:"varint,5,opt,name=spouse,proto3" json:"spouse,omitempty"`
	Money    float64 `protobuf:"fixed64,6,opt,name=money,proto3" json:"money,omitempty"`
}

func (x *GoogleGenerated) Reset() {
	*x = GoogleGenerated{}
	if protoimpl.UnsafeEnabled {
		mi := &file_structdef_go_v2_proto_msgTypes[0]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *GoogleGenerated) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*GoogleGenerated) ProtoMessage() {}

func (x *GoogleGenerated) ProtoReflect() protoreflect.Message {
	mi := &file_structdef_go_v2_proto_msgTypes[0]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use GoogleGenerated.ProtoReflect.Descriptor instead.
func (*GoogleGenerated) Descriptor() ([]byte, []int) {
	return file_structdef_go_v2_proto_rawDescGZIP(), []int{0}
}

func (x *GoogleGenerated) GetName() string {
	if x != nil {
		return x.Name
	}
	return ""
}

func (x *GoogleGenerated) GetBirthDay() int64 {
	if x != nil {
		return x.BirthDay
	}
	return 0
}

func (x *GoogleGenerated) GetPhone() string {
	if x != nil {
		return x.Phone
	}
	return ""
}

func (x *GoogleGenerated) GetSiblings() int32 {
	if x != nil {
		return x.Siblings
	}
	return 0
}

func (x *GoogleGenerated) GetSpouse() bool {
	if x != nil {
		return x.Spouse
	}
	return false
}

func (x *GoogleGenerated) GetMoney() float64 {
	if x != nil {
		return x.Money
	}
	return 0
}

var File_structdef_go_v2_proto protoreflect.FileDescriptor

var file_structdef_go_v2_proto_rawDesc = []byte{
	0x0a, 0x15, 0x73, 0x74, 0x72, 0x75, 0x63, 0x74, 0x64, 0x65, 0x66, 0x2d, 0x67, 0x6f, 0x2d, 0x76,
	0x32, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x12, 0x0a, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x62, 0x65,
	0x6e, 0x63, 0x68, 0x22, 0x96, 0x01, 0x0a, 0x04, 0x47, 0x6f, 0x56, 0x32, 0x12, 0x12, 0x0a, 0x04,
	0x6e, 0x61, 0x6d, 0x65, 0x18, 0x01, 0x20, 0x01, 0x28, 0x09, 0x52, 0x04, 0x6e, 0x61, 0x6d, 0x65,
	0x12, 0x1a, 0x0a, 0x08, 0x62, 0x69, 0x72, 0x74, 0x68, 0x44, 0x61, 0x79, 0x18, 0x02, 0x20, 0x01,
	0x28, 0x03, 0x52, 0x08, 0x62, 0x69, 0x72, 0x74, 0x68, 0x44, 0x61, 0x79, 0x12, 0x14, 0x0a, 0x05,
	0x70, 0x68, 0x6f, 0x6e, 0x65, 0x18, 0x03, 0x20, 0x01, 0x28, 0x09, 0x52, 0x05, 0x70, 0x68, 0x6f,
	0x6e, 0x65, 0x12, 0x1a, 0x0a, 0x08, 0x73, 0x69, 0x62, 0x6c, 0x69, 0x6e, 0x67, 0x73, 0x18, 0x04,
	0x20, 0x01, 0x28, 0x05, 0x52, 0x08, 0x73, 0x69, 0x62, 0x6c, 0x69, 0x6e, 0x67, 0x73, 0x12, 0x16,
	0x0a, 0x06, 0x73, 0x70, 0x6f, 0x75, 0x73, 0x65, 0x18, 0x05, 0x20, 0x01, 0x28, 0x08, 0x52, 0x06,
	0x73, 0x70, 0x6f, 0x75, 0x73, 0x65, 0x12, 0x14, 0x0a, 0x05, 0x6d, 0x6f, 0x6e, 0x65, 0x79, 0x18,
	0x06, 0x20, 0x01, 0x28, 0x01, 0x52, 0x05, 0x6d, 0x6f, 0x6e, 0x65, 0x79, 0x42, 0x2c, 0x5a, 0x2a,
	0x67, 0x69, 0x74, 0x68, 0x75, 0x62, 0x2e, 0x63, 0x6f, 0x6d, 0x2f, 0x61, 0x6c, 0x65, 0x78, 0x73,
	0x68, 0x74, 0x69, 0x6e, 0x2f, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x62, 0x65, 0x6e, 0x63, 0x68, 0x3b,
	0x70, 0x72, 0x6f, 0x74, 0x6f, 0x62, 0x65, 0x6e, 0x63, 0x68, 0x62, 0x06, 0x70, 0x72, 0x6f, 0x74,
	0x6f, 0x33,
}

var (
	file_structdef_go_v2_proto_rawDescOnce sync.Once
	file_structdef_go_v2_proto_rawDescData = file_structdef_go_v2_proto_rawDesc
)

func file_structdef_go_v2_proto_rawDescGZIP() []byte {
	file_structdef_go_v2_proto_rawDescOnce.Do(func() {
		file_structdef_go_v2_proto_rawDescData = protoimpl.X.CompressGZIP(file_structdef_go_v2_proto_rawDescData)
	})
	return file_structdef_go_v2_proto_rawDescData
}

var file_structdef_go_v2_proto_msgTypes = make([]protoimpl.MessageInfo, 1)
var file_structdef_go_v2_proto_goTypes = []interface{}{
	(*GoogleGenerated)(nil), // 0: protobench.GoogleGenerated
}
var file_structdef_go_v2_proto_depIdxs = []int32{
	0, // [0:0] is the sub-list for method output_type
	0, // [0:0] is the sub-list for method input_type
	0, // [0:0] is the sub-list for extension type_name
	0, // [0:0] is the sub-list for extension extendee
	0, // [0:0] is the sub-list for field type_name
}

func init() { file_structdef_go_v2_proto_init() }
func file_structdef_go_v2_proto_init() {
	if File_structdef_go_v2_proto != nil {
		return
	}
	if !protoimpl.UnsafeEnabled {
		file_structdef_go_v2_proto_msgTypes[0].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*GoogleGenerated); i {
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
			RawDescriptor: file_structdef_go_v2_proto_rawDesc,
			NumEnums:      0,
			NumMessages:   1,
			NumExtensions: 0,
			NumServices:   0,
		},
		GoTypes:           file_structdef_go_v2_proto_goTypes,
		DependencyIndexes: file_structdef_go_v2_proto_depIdxs,
		MessageInfos:      file_structdef_go_v2_proto_msgTypes,
	}.Build()
	File_structdef_go_v2_proto = out.File
	file_structdef_go_v2_proto_rawDesc = nil
	file_structdef_go_v2_proto_goTypes = nil
	file_structdef_go_v2_proto_depIdxs = nil
}
