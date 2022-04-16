// Code generated by protoc-gen-go. DO NOT EDIT.
// versions:
// 	protoc-gen-go v1.28.0
// 	protoc        v3.19.3
// source: grpc/regression/request.proto

package regression

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

type TestCaseReq struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	Captured   int64         `protobuf:"varint,1,opt,name=Captured,proto3" json:"Captured,omitempty"`
	AppID      string        `protobuf:"bytes,2,opt,name=AppID,proto3" json:"AppID,omitempty"`
	URI        string        `protobuf:"bytes,3,opt,name=URI,proto3" json:"URI,omitempty"`
	HttpReq    *HttpReq      `protobuf:"bytes,4,opt,name=HttpReq,proto3" json:"HttpReq,omitempty"`
	HttpResp   *HttpResp     `protobuf:"bytes,5,opt,name=HttpResp,proto3" json:"HttpResp,omitempty"`
	Dependency []*Dependency `protobuf:"bytes,6,rep,name=Dependency,proto3" json:"Dependency,omitempty"`
}

func (x *TestCaseReq) Reset() {
	*x = TestCaseReq{}
	if protoimpl.UnsafeEnabled {
		mi := &file_grpc_regression_request_proto_msgTypes[0]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *TestCaseReq) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*TestCaseReq) ProtoMessage() {}

func (x *TestCaseReq) ProtoReflect() protoreflect.Message {
	mi := &file_grpc_regression_request_proto_msgTypes[0]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use TestCaseReq.ProtoReflect.Descriptor instead.
func (*TestCaseReq) Descriptor() ([]byte, []int) {
	return file_grpc_regression_request_proto_rawDescGZIP(), []int{0}
}

func (x *TestCaseReq) GetCaptured() int64 {
	if x != nil {
		return x.Captured
	}
	return 0
}

func (x *TestCaseReq) GetAppID() string {
	if x != nil {
		return x.AppID
	}
	return ""
}

func (x *TestCaseReq) GetURI() string {
	if x != nil {
		return x.URI
	}
	return ""
}

func (x *TestCaseReq) GetHttpReq() *HttpReq {
	if x != nil {
		return x.HttpReq
	}
	return nil
}

func (x *TestCaseReq) GetHttpResp() *HttpResp {
	if x != nil {
		return x.HttpResp
	}
	return nil
}

func (x *TestCaseReq) GetDependency() []*Dependency {
	if x != nil {
		return x.Dependency
	}
	return nil
}

type TestReq struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	ID    string    `protobuf:"bytes,1,opt,name=ID,proto3" json:"ID,omitempty"`
	AppID string    `protobuf:"bytes,2,opt,name=AppID,proto3" json:"AppID,omitempty"`
	RunID string    `protobuf:"bytes,3,opt,name=RunID,proto3" json:"RunID,omitempty"`
	Resp  *HttpResp `protobuf:"bytes,4,opt,name=Resp,proto3" json:"Resp,omitempty"`
}

func (x *TestReq) Reset() {
	*x = TestReq{}
	if protoimpl.UnsafeEnabled {
		mi := &file_grpc_regression_request_proto_msgTypes[1]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *TestReq) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*TestReq) ProtoMessage() {}

func (x *TestReq) ProtoReflect() protoreflect.Message {
	mi := &file_grpc_regression_request_proto_msgTypes[1]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use TestReq.ProtoReflect.Descriptor instead.
func (*TestReq) Descriptor() ([]byte, []int) {
	return file_grpc_regression_request_proto_rawDescGZIP(), []int{1}
}

func (x *TestReq) GetID() string {
	if x != nil {
		return x.ID
	}
	return ""
}

func (x *TestReq) GetAppID() string {
	if x != nil {
		return x.AppID
	}
	return ""
}

func (x *TestReq) GetRunID() string {
	if x != nil {
		return x.RunID
	}
	return ""
}

func (x *TestReq) GetResp() *HttpResp {
	if x != nil {
		return x.Resp
	}
	return nil
}

type TestCase struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	Id       string             `protobuf:"bytes,1,opt,name=id,proto3" json:"id,omitempty"`
	Created  int64              `protobuf:"varint,2,opt,name=created,proto3" json:"created,omitempty"`
	Updated  int64              `protobuf:"varint,3,opt,name=updated,proto3" json:"updated,omitempty"`
	Captured int64              `protobuf:"varint,4,opt,name=captured,proto3" json:"captured,omitempty"`
	CID      string             `protobuf:"bytes,5,opt,name=CID,proto3" json:"CID,omitempty"`
	AppID    string             `protobuf:"bytes,6,opt,name=appID,proto3" json:"appID,omitempty"`
	URI      string             `protobuf:"bytes,7,opt,name=URI,proto3" json:"URI,omitempty"`
	HttpReq  *HttpReq           `protobuf:"bytes,8,opt,name=HttpReq,proto3" json:"HttpReq,omitempty"`
	HttpResp *HttpResp          `protobuf:"bytes,9,opt,name=HttpResp,proto3" json:"HttpResp,omitempty"`
	Deps     []*Dependency      `protobuf:"bytes,10,rep,name=Deps,proto3" json:"Deps,omitempty"`
	AllKeys  map[string]*StrArr `protobuf:"bytes,11,rep,name=allKeys,proto3" json:"allKeys,omitempty" protobuf_key:"bytes,1,opt,name=key,proto3" protobuf_val:"bytes,2,opt,name=value,proto3"`
	Anchors  map[string]*StrArr `protobuf:"bytes,12,rep,name=anchors,proto3" json:"anchors,omitempty" protobuf_key:"bytes,1,opt,name=key,proto3" protobuf_val:"bytes,2,opt,name=value,proto3"`
	Noise    []string           `protobuf:"bytes,13,rep,name=noise,proto3" json:"noise,omitempty"`
}

func (x *TestCase) Reset() {
	*x = TestCase{}
	if protoimpl.UnsafeEnabled {
		mi := &file_grpc_regression_request_proto_msgTypes[2]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *TestCase) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*TestCase) ProtoMessage() {}

func (x *TestCase) ProtoReflect() protoreflect.Message {
	mi := &file_grpc_regression_request_proto_msgTypes[2]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use TestCase.ProtoReflect.Descriptor instead.
func (*TestCase) Descriptor() ([]byte, []int) {
	return file_grpc_regression_request_proto_rawDescGZIP(), []int{2}
}

func (x *TestCase) GetId() string {
	if x != nil {
		return x.Id
	}
	return ""
}

func (x *TestCase) GetCreated() int64 {
	if x != nil {
		return x.Created
	}
	return 0
}

func (x *TestCase) GetUpdated() int64 {
	if x != nil {
		return x.Updated
	}
	return 0
}

func (x *TestCase) GetCaptured() int64 {
	if x != nil {
		return x.Captured
	}
	return 0
}

func (x *TestCase) GetCID() string {
	if x != nil {
		return x.CID
	}
	return ""
}

func (x *TestCase) GetAppID() string {
	if x != nil {
		return x.AppID
	}
	return ""
}

func (x *TestCase) GetURI() string {
	if x != nil {
		return x.URI
	}
	return ""
}

func (x *TestCase) GetHttpReq() *HttpReq {
	if x != nil {
		return x.HttpReq
	}
	return nil
}

func (x *TestCase) GetHttpResp() *HttpResp {
	if x != nil {
		return x.HttpResp
	}
	return nil
}

func (x *TestCase) GetDeps() []*Dependency {
	if x != nil {
		return x.Deps
	}
	return nil
}

func (x *TestCase) GetAllKeys() map[string]*StrArr {
	if x != nil {
		return x.AllKeys
	}
	return nil
}

func (x *TestCase) GetAnchors() map[string]*StrArr {
	if x != nil {
		return x.Anchors
	}
	return nil
}

func (x *TestCase) GetNoise() []string {
	if x != nil {
		return x.Noise
	}
	return nil
}

var File_grpc_regression_request_proto protoreflect.FileDescriptor

var file_grpc_regression_request_proto_rawDesc = []byte{
	0x0a, 0x1d, 0x67, 0x72, 0x70, 0x63, 0x2f, 0x72, 0x65, 0x67, 0x72, 0x65, 0x73, 0x73, 0x69, 0x6f,
	0x6e, 0x2f, 0x72, 0x65, 0x71, 0x75, 0x65, 0x73, 0x74, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x12,
	0x05, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x1a, 0x21, 0x67, 0x72, 0x70, 0x63, 0x2f, 0x72, 0x65, 0x67,
	0x72, 0x65, 0x73, 0x73, 0x69, 0x6f, 0x6e, 0x2f, 0x6d, 0x6f, 0x64, 0x65, 0x6c, 0x73, 0x2e, 0x68,
	0x74, 0x74, 0x70, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x1a, 0x20, 0x67, 0x72, 0x70, 0x63, 0x2f,
	0x72, 0x65, 0x67, 0x72, 0x65, 0x73, 0x73, 0x69, 0x6f, 0x6e, 0x2f, 0x6d, 0x6f, 0x64, 0x65, 0x6c,
	0x73, 0x2e, 0x64, 0x65, 0x70, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x22, 0xde, 0x01, 0x0a, 0x0b,
	0x54, 0x65, 0x73, 0x74, 0x43, 0x61, 0x73, 0x65, 0x52, 0x65, 0x71, 0x12, 0x1a, 0x0a, 0x08, 0x43,
	0x61, 0x70, 0x74, 0x75, 0x72, 0x65, 0x64, 0x18, 0x01, 0x20, 0x01, 0x28, 0x03, 0x52, 0x08, 0x43,
	0x61, 0x70, 0x74, 0x75, 0x72, 0x65, 0x64, 0x12, 0x14, 0x0a, 0x05, 0x41, 0x70, 0x70, 0x49, 0x44,
	0x18, 0x02, 0x20, 0x01, 0x28, 0x09, 0x52, 0x05, 0x41, 0x70, 0x70, 0x49, 0x44, 0x12, 0x10, 0x0a,
	0x03, 0x55, 0x52, 0x49, 0x18, 0x03, 0x20, 0x01, 0x28, 0x09, 0x52, 0x03, 0x55, 0x52, 0x49, 0x12,
	0x29, 0x0a, 0x07, 0x48, 0x74, 0x74, 0x70, 0x52, 0x65, 0x71, 0x18, 0x04, 0x20, 0x01, 0x28, 0x0b,
	0x32, 0x0f, 0x2e, 0x6d, 0x6f, 0x64, 0x65, 0x6c, 0x73, 0x2e, 0x48, 0x74, 0x74, 0x70, 0x52, 0x65,
	0x71, 0x52, 0x07, 0x48, 0x74, 0x74, 0x70, 0x52, 0x65, 0x71, 0x12, 0x2c, 0x0a, 0x08, 0x48, 0x74,
	0x74, 0x70, 0x52, 0x65, 0x73, 0x70, 0x18, 0x05, 0x20, 0x01, 0x28, 0x0b, 0x32, 0x10, 0x2e, 0x6d,
	0x6f, 0x64, 0x65, 0x6c, 0x73, 0x2e, 0x48, 0x74, 0x74, 0x70, 0x52, 0x65, 0x73, 0x70, 0x52, 0x08,
	0x48, 0x74, 0x74, 0x70, 0x52, 0x65, 0x73, 0x70, 0x12, 0x32, 0x0a, 0x0a, 0x44, 0x65, 0x70, 0x65,
	0x6e, 0x64, 0x65, 0x6e, 0x63, 0x79, 0x18, 0x06, 0x20, 0x03, 0x28, 0x0b, 0x32, 0x12, 0x2e, 0x6d,
	0x6f, 0x64, 0x65, 0x6c, 0x73, 0x2e, 0x44, 0x65, 0x70, 0x65, 0x6e, 0x64, 0x65, 0x6e, 0x63, 0x79,
	0x52, 0x0a, 0x44, 0x65, 0x70, 0x65, 0x6e, 0x64, 0x65, 0x6e, 0x63, 0x79, 0x22, 0x6b, 0x0a, 0x07,
	0x54, 0x65, 0x73, 0x74, 0x52, 0x65, 0x71, 0x12, 0x0e, 0x0a, 0x02, 0x49, 0x44, 0x18, 0x01, 0x20,
	0x01, 0x28, 0x09, 0x52, 0x02, 0x49, 0x44, 0x12, 0x14, 0x0a, 0x05, 0x41, 0x70, 0x70, 0x49, 0x44,
	0x18, 0x02, 0x20, 0x01, 0x28, 0x09, 0x52, 0x05, 0x41, 0x70, 0x70, 0x49, 0x44, 0x12, 0x14, 0x0a,
	0x05, 0x52, 0x75, 0x6e, 0x49, 0x44, 0x18, 0x03, 0x20, 0x01, 0x28, 0x09, 0x52, 0x05, 0x52, 0x75,
	0x6e, 0x49, 0x44, 0x12, 0x24, 0x0a, 0x04, 0x52, 0x65, 0x73, 0x70, 0x18, 0x04, 0x20, 0x01, 0x28,
	0x0b, 0x32, 0x10, 0x2e, 0x6d, 0x6f, 0x64, 0x65, 0x6c, 0x73, 0x2e, 0x48, 0x74, 0x74, 0x70, 0x52,
	0x65, 0x73, 0x70, 0x52, 0x04, 0x52, 0x65, 0x73, 0x70, 0x22, 0xc3, 0x04, 0x0a, 0x08, 0x54, 0x65,
	0x73, 0x74, 0x43, 0x61, 0x73, 0x65, 0x12, 0x0e, 0x0a, 0x02, 0x69, 0x64, 0x18, 0x01, 0x20, 0x01,
	0x28, 0x09, 0x52, 0x02, 0x69, 0x64, 0x12, 0x18, 0x0a, 0x07, 0x63, 0x72, 0x65, 0x61, 0x74, 0x65,
	0x64, 0x18, 0x02, 0x20, 0x01, 0x28, 0x03, 0x52, 0x07, 0x63, 0x72, 0x65, 0x61, 0x74, 0x65, 0x64,
	0x12, 0x18, 0x0a, 0x07, 0x75, 0x70, 0x64, 0x61, 0x74, 0x65, 0x64, 0x18, 0x03, 0x20, 0x01, 0x28,
	0x03, 0x52, 0x07, 0x75, 0x70, 0x64, 0x61, 0x74, 0x65, 0x64, 0x12, 0x1a, 0x0a, 0x08, 0x63, 0x61,
	0x70, 0x74, 0x75, 0x72, 0x65, 0x64, 0x18, 0x04, 0x20, 0x01, 0x28, 0x03, 0x52, 0x08, 0x63, 0x61,
	0x70, 0x74, 0x75, 0x72, 0x65, 0x64, 0x12, 0x10, 0x0a, 0x03, 0x43, 0x49, 0x44, 0x18, 0x05, 0x20,
	0x01, 0x28, 0x09, 0x52, 0x03, 0x43, 0x49, 0x44, 0x12, 0x14, 0x0a, 0x05, 0x61, 0x70, 0x70, 0x49,
	0x44, 0x18, 0x06, 0x20, 0x01, 0x28, 0x09, 0x52, 0x05, 0x61, 0x70, 0x70, 0x49, 0x44, 0x12, 0x10,
	0x0a, 0x03, 0x55, 0x52, 0x49, 0x18, 0x07, 0x20, 0x01, 0x28, 0x09, 0x52, 0x03, 0x55, 0x52, 0x49,
	0x12, 0x29, 0x0a, 0x07, 0x48, 0x74, 0x74, 0x70, 0x52, 0x65, 0x71, 0x18, 0x08, 0x20, 0x01, 0x28,
	0x0b, 0x32, 0x0f, 0x2e, 0x6d, 0x6f, 0x64, 0x65, 0x6c, 0x73, 0x2e, 0x48, 0x74, 0x74, 0x70, 0x52,
	0x65, 0x71, 0x52, 0x07, 0x48, 0x74, 0x74, 0x70, 0x52, 0x65, 0x71, 0x12, 0x2c, 0x0a, 0x08, 0x48,
	0x74, 0x74, 0x70, 0x52, 0x65, 0x73, 0x70, 0x18, 0x09, 0x20, 0x01, 0x28, 0x0b, 0x32, 0x10, 0x2e,
	0x6d, 0x6f, 0x64, 0x65, 0x6c, 0x73, 0x2e, 0x48, 0x74, 0x74, 0x70, 0x52, 0x65, 0x73, 0x70, 0x52,
	0x08, 0x48, 0x74, 0x74, 0x70, 0x52, 0x65, 0x73, 0x70, 0x12, 0x26, 0x0a, 0x04, 0x44, 0x65, 0x70,
	0x73, 0x18, 0x0a, 0x20, 0x03, 0x28, 0x0b, 0x32, 0x12, 0x2e, 0x6d, 0x6f, 0x64, 0x65, 0x6c, 0x73,
	0x2e, 0x44, 0x65, 0x70, 0x65, 0x6e, 0x64, 0x65, 0x6e, 0x63, 0x79, 0x52, 0x04, 0x44, 0x65, 0x70,
	0x73, 0x12, 0x36, 0x0a, 0x07, 0x61, 0x6c, 0x6c, 0x4b, 0x65, 0x79, 0x73, 0x18, 0x0b, 0x20, 0x03,
	0x28, 0x0b, 0x32, 0x1c, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x2e, 0x54, 0x65, 0x73, 0x74, 0x43,
	0x61, 0x73, 0x65, 0x2e, 0x41, 0x6c, 0x6c, 0x4b, 0x65, 0x79, 0x73, 0x45, 0x6e, 0x74, 0x72, 0x79,
	0x52, 0x07, 0x61, 0x6c, 0x6c, 0x4b, 0x65, 0x79, 0x73, 0x12, 0x36, 0x0a, 0x07, 0x61, 0x6e, 0x63,
	0x68, 0x6f, 0x72, 0x73, 0x18, 0x0c, 0x20, 0x03, 0x28, 0x0b, 0x32, 0x1c, 0x2e, 0x70, 0x72, 0x6f,
	0x74, 0x6f, 0x2e, 0x54, 0x65, 0x73, 0x74, 0x43, 0x61, 0x73, 0x65, 0x2e, 0x41, 0x6e, 0x63, 0x68,
	0x6f, 0x72, 0x73, 0x45, 0x6e, 0x74, 0x72, 0x79, 0x52, 0x07, 0x61, 0x6e, 0x63, 0x68, 0x6f, 0x72,
	0x73, 0x12, 0x14, 0x0a, 0x05, 0x6e, 0x6f, 0x69, 0x73, 0x65, 0x18, 0x0d, 0x20, 0x03, 0x28, 0x09,
	0x52, 0x05, 0x6e, 0x6f, 0x69, 0x73, 0x65, 0x1a, 0x4a, 0x0a, 0x0c, 0x41, 0x6c, 0x6c, 0x4b, 0x65,
	0x79, 0x73, 0x45, 0x6e, 0x74, 0x72, 0x79, 0x12, 0x10, 0x0a, 0x03, 0x6b, 0x65, 0x79, 0x18, 0x01,
	0x20, 0x01, 0x28, 0x09, 0x52, 0x03, 0x6b, 0x65, 0x79, 0x12, 0x24, 0x0a, 0x05, 0x76, 0x61, 0x6c,
	0x75, 0x65, 0x18, 0x02, 0x20, 0x01, 0x28, 0x0b, 0x32, 0x0e, 0x2e, 0x6d, 0x6f, 0x64, 0x65, 0x6c,
	0x73, 0x2e, 0x53, 0x74, 0x72, 0x41, 0x72, 0x72, 0x52, 0x05, 0x76, 0x61, 0x6c, 0x75, 0x65, 0x3a,
	0x02, 0x38, 0x01, 0x1a, 0x4a, 0x0a, 0x0c, 0x41, 0x6e, 0x63, 0x68, 0x6f, 0x72, 0x73, 0x45, 0x6e,
	0x74, 0x72, 0x79, 0x12, 0x10, 0x0a, 0x03, 0x6b, 0x65, 0x79, 0x18, 0x01, 0x20, 0x01, 0x28, 0x09,
	0x52, 0x03, 0x6b, 0x65, 0x79, 0x12, 0x24, 0x0a, 0x05, 0x76, 0x61, 0x6c, 0x75, 0x65, 0x18, 0x02,
	0x20, 0x01, 0x28, 0x0b, 0x32, 0x0e, 0x2e, 0x6d, 0x6f, 0x64, 0x65, 0x6c, 0x73, 0x2e, 0x53, 0x74,
	0x72, 0x41, 0x72, 0x72, 0x52, 0x05, 0x76, 0x61, 0x6c, 0x75, 0x65, 0x3a, 0x02, 0x38, 0x01, 0x42,
	0x25, 0x5a, 0x23, 0x67, 0x6f, 0x2e, 0x6b, 0x65, 0x70, 0x6c, 0x6f, 0x79, 0x2e, 0x69, 0x6f, 0x2f,
	0x73, 0x65, 0x72, 0x76, 0x65, 0x72, 0x2f, 0x67, 0x72, 0x70, 0x63, 0x2f, 0x72, 0x65, 0x67, 0x72,
	0x65, 0x73, 0x73, 0x69, 0x6f, 0x6e, 0x50, 0x00, 0x62, 0x06, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x33,
}

var (
	file_grpc_regression_request_proto_rawDescOnce sync.Once
	file_grpc_regression_request_proto_rawDescData = file_grpc_regression_request_proto_rawDesc
)

func file_grpc_regression_request_proto_rawDescGZIP() []byte {
	file_grpc_regression_request_proto_rawDescOnce.Do(func() {
		file_grpc_regression_request_proto_rawDescData = protoimpl.X.CompressGZIP(file_grpc_regression_request_proto_rawDescData)
	})
	return file_grpc_regression_request_proto_rawDescData
}

var file_grpc_regression_request_proto_msgTypes = make([]protoimpl.MessageInfo, 5)
var file_grpc_regression_request_proto_goTypes = []interface{}{
	(*TestCaseReq)(nil), // 0: proto.TestCaseReq
	(*TestReq)(nil),     // 1: proto.TestReq
	(*TestCase)(nil),    // 2: proto.TestCase
	nil,                 // 3: proto.TestCase.AllKeysEntry
	nil,                 // 4: proto.TestCase.AnchorsEntry
	(*HttpReq)(nil),     // 5: models.HttpReq
	(*HttpResp)(nil),    // 6: models.HttpResp
	(*Dependency)(nil),  // 7: models.Dependency
	(*StrArr)(nil),      // 8: models.StrArr
}
var file_grpc_regression_request_proto_depIdxs = []int32{
	5,  // 0: proto.TestCaseReq.HttpReq:type_name -> models.HttpReq
	6,  // 1: proto.TestCaseReq.HttpResp:type_name -> models.HttpResp
	7,  // 2: proto.TestCaseReq.Dependency:type_name -> models.Dependency
	6,  // 3: proto.TestReq.Resp:type_name -> models.HttpResp
	5,  // 4: proto.TestCase.HttpReq:type_name -> models.HttpReq
	6,  // 5: proto.TestCase.HttpResp:type_name -> models.HttpResp
	7,  // 6: proto.TestCase.Deps:type_name -> models.Dependency
	3,  // 7: proto.TestCase.allKeys:type_name -> proto.TestCase.AllKeysEntry
	4,  // 8: proto.TestCase.anchors:type_name -> proto.TestCase.AnchorsEntry
	8,  // 9: proto.TestCase.AllKeysEntry.value:type_name -> models.StrArr
	8,  // 10: proto.TestCase.AnchorsEntry.value:type_name -> models.StrArr
	11, // [11:11] is the sub-list for method output_type
	11, // [11:11] is the sub-list for method input_type
	11, // [11:11] is the sub-list for extension type_name
	11, // [11:11] is the sub-list for extension extendee
	0,  // [0:11] is the sub-list for field type_name
}

func init() { file_grpc_regression_request_proto_init() }
func file_grpc_regression_request_proto_init() {
	if File_grpc_regression_request_proto != nil {
		return
	}
	file_grpc_regression_models_http_proto_init()
	file_grpc_regression_models_dep_proto_init()
	if !protoimpl.UnsafeEnabled {
		file_grpc_regression_request_proto_msgTypes[0].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*TestCaseReq); i {
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
		file_grpc_regression_request_proto_msgTypes[1].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*TestReq); i {
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
		file_grpc_regression_request_proto_msgTypes[2].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*TestCase); i {
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
			RawDescriptor: file_grpc_regression_request_proto_rawDesc,
			NumEnums:      0,
			NumMessages:   5,
			NumExtensions: 0,
			NumServices:   0,
		},
		GoTypes:           file_grpc_regression_request_proto_goTypes,
		DependencyIndexes: file_grpc_regression_request_proto_depIdxs,
		MessageInfos:      file_grpc_regression_request_proto_msgTypes,
	}.Build()
	File_grpc_regression_request_proto = out.File
	file_grpc_regression_request_proto_rawDesc = nil
	file_grpc_regression_request_proto_goTypes = nil
	file_grpc_regression_request_proto_depIdxs = nil
}