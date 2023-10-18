// Code generated by protoc-gen-go. DO NOT EDIT.
// versions:
// 	protoc-gen-go v1.30.0
// 	protoc        v3.21.12
// source: locks.proto

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

type BalanceLockRequest struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	Id      string   `protobuf:"bytes,1,opt,name=id,proto3" json:"id,omitempty"`           // lock identifier ( optional parameter, if not specified - txID is used)
	Address string   `protobuf:"bytes,2,opt,name=address,proto3" json:"address,omitempty"` // owner address
	Token   string   `protobuf:"bytes,3,opt,name=token,proto3" json:"token,omitempty"`     // token identifier/ticker
	Amount  string   `protobuf:"bytes,4,opt,name=amount,proto3" json:"amount,omitempty"`   // big.Int number of tokens to block
	Reason  string   `protobuf:"bytes,5,opt,name=reason,proto3" json:"reason,omitempty"`   // reason for locking
	Docs    [][]byte `protobuf:"bytes,6,rep,name=docs,proto3" json:"docs,omitempty"`       // hashes of documents with justification (optional parameter)
	Payload []byte   `protobuf:"bytes,7,opt,name=payload,proto3" json:"payload,omitempty"` // additional information (optional parameter)
}

func (x *BalanceLockRequest) Reset() {
	*x = BalanceLockRequest{}
	if protoimpl.UnsafeEnabled {
		mi := &file_locks_proto_msgTypes[0]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *BalanceLockRequest) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*BalanceLockRequest) ProtoMessage() {}

func (x *BalanceLockRequest) ProtoReflect() protoreflect.Message {
	mi := &file_locks_proto_msgTypes[0]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use BalanceLockRequest.ProtoReflect.Descriptor instead.
func (*BalanceLockRequest) Descriptor() ([]byte, []int) {
	return file_locks_proto_rawDescGZIP(), []int{0}
}

func (x *BalanceLockRequest) GetId() string {
	if x != nil {
		return x.Id
	}
	return ""
}

func (x *BalanceLockRequest) GetAddress() string {
	if x != nil {
		return x.Address
	}
	return ""
}

func (x *BalanceLockRequest) GetToken() string {
	if x != nil {
		return x.Token
	}
	return ""
}

func (x *BalanceLockRequest) GetAmount() string {
	if x != nil {
		return x.Amount
	}
	return ""
}

func (x *BalanceLockRequest) GetReason() string {
	if x != nil {
		return x.Reason
	}
	return ""
}

func (x *BalanceLockRequest) GetDocs() [][]byte {
	if x != nil {
		return x.Docs
	}
	return nil
}

func (x *BalanceLockRequest) GetPayload() []byte {
	if x != nil {
		return x.Payload
	}
	return nil
}

// State: balance token locking data
type TokenBalanceLock struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	Id            string   `protobuf:"bytes,1,opt,name=id,proto3" json:"id,omitempty"`                                            // lock identifier (optional parameter, if not specified - txID is used)
	Address       string   `protobuf:"bytes,2,opt,name=address,proto3" json:"address,omitempty"`                                  // owner address
	Token         string   `protobuf:"bytes,3,opt,name=token,proto3" json:"token,omitempty"`                                      // token identifier/ticker
	InitAmount    string   `protobuf:"bytes,4,opt,name=init_amount,json=initAmount,proto3" json:"init_amount,omitempty"`          // big.Int initial number of tokens to block
	CurrentAmount string   `protobuf:"bytes,5,opt,name=current_amount,json=currentAmount,proto3" json:"current_amount,omitempty"` // big.Int current number of tokens to be blocked
	Reason        string   `protobuf:"bytes,6,opt,name=reason,proto3" json:"reason,omitempty"`                                    // reason for blocking
	Docs          [][]byte `protobuf:"bytes,7,rep,name=docs,proto3" json:"docs,omitempty"`                                        // hashes of documents with justification (optional parameter)
	Payload       []byte   `protobuf:"bytes,8,opt,name=payload,proto3" json:"payload,omitempty"`                                  // additional information (optional parameter)
}

func (x *TokenBalanceLock) Reset() {
	*x = TokenBalanceLock{}
	if protoimpl.UnsafeEnabled {
		mi := &file_locks_proto_msgTypes[1]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *TokenBalanceLock) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*TokenBalanceLock) ProtoMessage() {}

func (x *TokenBalanceLock) ProtoReflect() protoreflect.Message {
	mi := &file_locks_proto_msgTypes[1]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use TokenBalanceLock.ProtoReflect.Descriptor instead.
func (*TokenBalanceLock) Descriptor() ([]byte, []int) {
	return file_locks_proto_rawDescGZIP(), []int{1}
}

func (x *TokenBalanceLock) GetId() string {
	if x != nil {
		return x.Id
	}
	return ""
}

func (x *TokenBalanceLock) GetAddress() string {
	if x != nil {
		return x.Address
	}
	return ""
}

func (x *TokenBalanceLock) GetToken() string {
	if x != nil {
		return x.Token
	}
	return ""
}

func (x *TokenBalanceLock) GetInitAmount() string {
	if x != nil {
		return x.InitAmount
	}
	return ""
}

func (x *TokenBalanceLock) GetCurrentAmount() string {
	if x != nil {
		return x.CurrentAmount
	}
	return ""
}

func (x *TokenBalanceLock) GetReason() string {
	if x != nil {
		return x.Reason
	}
	return ""
}

func (x *TokenBalanceLock) GetDocs() [][]byte {
	if x != nil {
		return x.Docs
	}
	return nil
}

func (x *TokenBalanceLock) GetPayload() []byte {
	if x != nil {
		return x.Payload
	}
	return nil
}

// State: allowedbalance lock data
type AllowedBalanceLock struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	Id            string   `protobuf:"bytes,1,opt,name=id,proto3" json:"id,omitempty"`                                            // lock identifier (optional parameter, if not specified - txID is used)
	Address       string   `protobuf:"bytes,2,opt,name=address,proto3" json:"address,omitempty"`                                  // owner address
	Token         string   `protobuf:"bytes,3,opt,name=token,proto3" json:"token,omitempty"`                                      // token identifier/ticker
	InitAmount    string   `protobuf:"bytes,4,opt,name=init_amount,json=initAmount,proto3" json:"init_amount,omitempty"`          // big.Int initial number of tokens to block
	CurrentAmount string   `protobuf:"bytes,5,opt,name=current_amount,json=currentAmount,proto3" json:"current_amount,omitempty"` // big.Int current number of tokens to be blocked
	Reason        string   `protobuf:"bytes,6,opt,name=reason,proto3" json:"reason,omitempty"`                                    // reason for blocking
	Docs          [][]byte `protobuf:"bytes,7,rep,name=docs,proto3" json:"docs,omitempty"`                                        // hashes of documents with justification (optional parameter)
	Payload       []byte   `protobuf:"bytes,8,opt,name=payload,proto3" json:"payload,omitempty"`                                  // additional information (optional parameter)
}

func (x *AllowedBalanceLock) Reset() {
	*x = AllowedBalanceLock{}
	if protoimpl.UnsafeEnabled {
		mi := &file_locks_proto_msgTypes[2]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *AllowedBalanceLock) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*AllowedBalanceLock) ProtoMessage() {}

func (x *AllowedBalanceLock) ProtoReflect() protoreflect.Message {
	mi := &file_locks_proto_msgTypes[2]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use AllowedBalanceLock.ProtoReflect.Descriptor instead.
func (*AllowedBalanceLock) Descriptor() ([]byte, []int) {
	return file_locks_proto_rawDescGZIP(), []int{2}
}

func (x *AllowedBalanceLock) GetId() string {
	if x != nil {
		return x.Id
	}
	return ""
}

func (x *AllowedBalanceLock) GetAddress() string {
	if x != nil {
		return x.Address
	}
	return ""
}

func (x *AllowedBalanceLock) GetToken() string {
	if x != nil {
		return x.Token
	}
	return ""
}

func (x *AllowedBalanceLock) GetInitAmount() string {
	if x != nil {
		return x.InitAmount
	}
	return ""
}

func (x *AllowedBalanceLock) GetCurrentAmount() string {
	if x != nil {
		return x.CurrentAmount
	}
	return ""
}

func (x *AllowedBalanceLock) GetReason() string {
	if x != nil {
		return x.Reason
	}
	return ""
}

func (x *AllowedBalanceLock) GetDocs() [][]byte {
	if x != nil {
		return x.Docs
	}
	return nil
}

func (x *AllowedBalanceLock) GetPayload() []byte {
	if x != nil {
		return x.Payload
	}
	return nil
}

// Event: token balance blocked
type TokenBalanceLocked struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	Id      string   `protobuf:"bytes,1,opt,name=id,proto3" json:"id,omitempty"`           // lock identifier
	Address string   `protobuf:"bytes,2,opt,name=address,proto3" json:"address,omitempty"` // owner address
	Token   string   `protobuf:"bytes,3,opt,name=token,proto3" json:"token,omitempty"`     // token identifier/ticker
	Amount  string   `protobuf:"bytes,4,opt,name=amount,proto3" json:"amount,omitempty"`   // big.Int number of tokens to block
	Reason  string   `protobuf:"bytes,5,opt,name=reason,proto3" json:"reason,omitempty"`   // reason for locking
	Docs    [][]byte `protobuf:"bytes,6,rep,name=docs,proto3" json:"docs,omitempty"`       // hashes of documents with justification (optional parameter)
	Payload []byte   `protobuf:"bytes,7,opt,name=payload,proto3" json:"payload,omitempty"` // additional information (optional parameter)
}

func (x *TokenBalanceLocked) Reset() {
	*x = TokenBalanceLocked{}
	if protoimpl.UnsafeEnabled {
		mi := &file_locks_proto_msgTypes[3]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *TokenBalanceLocked) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*TokenBalanceLocked) ProtoMessage() {}

func (x *TokenBalanceLocked) ProtoReflect() protoreflect.Message {
	mi := &file_locks_proto_msgTypes[3]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use TokenBalanceLocked.ProtoReflect.Descriptor instead.
func (*TokenBalanceLocked) Descriptor() ([]byte, []int) {
	return file_locks_proto_rawDescGZIP(), []int{3}
}

func (x *TokenBalanceLocked) GetId() string {
	if x != nil {
		return x.Id
	}
	return ""
}

func (x *TokenBalanceLocked) GetAddress() string {
	if x != nil {
		return x.Address
	}
	return ""
}

func (x *TokenBalanceLocked) GetToken() string {
	if x != nil {
		return x.Token
	}
	return ""
}

func (x *TokenBalanceLocked) GetAmount() string {
	if x != nil {
		return x.Amount
	}
	return ""
}

func (x *TokenBalanceLocked) GetReason() string {
	if x != nil {
		return x.Reason
	}
	return ""
}

func (x *TokenBalanceLocked) GetDocs() [][]byte {
	if x != nil {
		return x.Docs
	}
	return nil
}

func (x *TokenBalanceLocked) GetPayload() []byte {
	if x != nil {
		return x.Payload
	}
	return nil
}

// Event: token balance unlocked
type TokenBalanceUnlocked struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	Id                string   `protobuf:"bytes,1,opt,name=id,proto3" json:"id,omitempty"`                                                         // lock identifier
	Address           string   `protobuf:"bytes,2,opt,name=address,proto3" json:"address,omitempty"`                                               // owner address
	Token             string   `protobuf:"bytes,3,opt,name=token,proto3" json:"token,omitempty"`                                                   // token identifier/ticker
	Amount            string   `protobuf:"bytes,4,opt,name=amount,proto3" json:"amount,omitempty"`                                                 // big.Int amount of tokens to unlock
	Reason            string   `protobuf:"bytes,5,opt,name=reason,proto3" json:"reason,omitempty"`                                                 // reason for locking
	Docs              [][]byte `protobuf:"bytes,6,rep,name=docs,proto3" json:"docs,omitempty"`                                                     // hashes of documents with justification (optional parameter)
	Payload           []byte   `protobuf:"bytes,7,opt,name=payload,proto3" json:"payload,omitempty"`                                               // additional information (optional parameter)
	CompleteOperation bool     `protobuf:"varint,8,opt,name=complete_operation,json=completeOperation,proto3" json:"complete_operation,omitempty"` // sign that it is completely unlocked
}

func (x *TokenBalanceUnlocked) Reset() {
	*x = TokenBalanceUnlocked{}
	if protoimpl.UnsafeEnabled {
		mi := &file_locks_proto_msgTypes[4]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *TokenBalanceUnlocked) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*TokenBalanceUnlocked) ProtoMessage() {}

func (x *TokenBalanceUnlocked) ProtoReflect() protoreflect.Message {
	mi := &file_locks_proto_msgTypes[4]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use TokenBalanceUnlocked.ProtoReflect.Descriptor instead.
func (*TokenBalanceUnlocked) Descriptor() ([]byte, []int) {
	return file_locks_proto_rawDescGZIP(), []int{4}
}

func (x *TokenBalanceUnlocked) GetId() string {
	if x != nil {
		return x.Id
	}
	return ""
}

func (x *TokenBalanceUnlocked) GetAddress() string {
	if x != nil {
		return x.Address
	}
	return ""
}

func (x *TokenBalanceUnlocked) GetToken() string {
	if x != nil {
		return x.Token
	}
	return ""
}

func (x *TokenBalanceUnlocked) GetAmount() string {
	if x != nil {
		return x.Amount
	}
	return ""
}

func (x *TokenBalanceUnlocked) GetReason() string {
	if x != nil {
		return x.Reason
	}
	return ""
}

func (x *TokenBalanceUnlocked) GetDocs() [][]byte {
	if x != nil {
		return x.Docs
	}
	return nil
}

func (x *TokenBalanceUnlocked) GetPayload() []byte {
	if x != nil {
		return x.Payload
	}
	return nil
}

func (x *TokenBalanceUnlocked) GetCompleteOperation() bool {
	if x != nil {
		return x.CompleteOperation
	}
	return false
}

// Event: balance token is locked
type AllowedBalanceLocked struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	Id      string   `protobuf:"bytes,1,opt,name=id,proto3" json:"id,omitempty"`           // lock identifier
	Address string   `protobuf:"bytes,2,opt,name=address,proto3" json:"address,omitempty"` // owner address
	Token   string   `protobuf:"bytes,3,opt,name=token,proto3" json:"token,omitempty"`     // token identifier/ticker
	Amount  string   `protobuf:"bytes,4,opt,name=amount,proto3" json:"amount,omitempty"`   // big.Int amount of tokens to unlock
	Reason  string   `protobuf:"bytes,5,opt,name=reason,proto3" json:"reason,omitempty"`   // reason for locking
	Docs    [][]byte `protobuf:"bytes,6,rep,name=docs,proto3" json:"docs,omitempty"`       // hashes of documents with justification (optional parameter)
	Payload []byte   `protobuf:"bytes,7,opt,name=payload,proto3" json:"payload,omitempty"` // additional information (optional parameter)
}

func (x *AllowedBalanceLocked) Reset() {
	*x = AllowedBalanceLocked{}
	if protoimpl.UnsafeEnabled {
		mi := &file_locks_proto_msgTypes[5]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *AllowedBalanceLocked) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*AllowedBalanceLocked) ProtoMessage() {}

func (x *AllowedBalanceLocked) ProtoReflect() protoreflect.Message {
	mi := &file_locks_proto_msgTypes[5]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use AllowedBalanceLocked.ProtoReflect.Descriptor instead.
func (*AllowedBalanceLocked) Descriptor() ([]byte, []int) {
	return file_locks_proto_rawDescGZIP(), []int{5}
}

func (x *AllowedBalanceLocked) GetId() string {
	if x != nil {
		return x.Id
	}
	return ""
}

func (x *AllowedBalanceLocked) GetAddress() string {
	if x != nil {
		return x.Address
	}
	return ""
}

func (x *AllowedBalanceLocked) GetToken() string {
	if x != nil {
		return x.Token
	}
	return ""
}

func (x *AllowedBalanceLocked) GetAmount() string {
	if x != nil {
		return x.Amount
	}
	return ""
}

func (x *AllowedBalanceLocked) GetReason() string {
	if x != nil {
		return x.Reason
	}
	return ""
}

func (x *AllowedBalanceLocked) GetDocs() [][]byte {
	if x != nil {
		return x.Docs
	}
	return nil
}

func (x *AllowedBalanceLocked) GetPayload() []byte {
	if x != nil {
		return x.Payload
	}
	return nil
}

// Event: token balance unlocked
type AllowedBalanceUnlocked struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	Id                string   `protobuf:"bytes,1,opt,name=id,proto3" json:"id,omitempty"`                                                         // lock identifier
	Address           string   `protobuf:"bytes,2,opt,name=address,proto3" json:"address,omitempty"`                                               // owner address
	Token             string   `protobuf:"bytes,3,opt,name=token,proto3" json:"token,omitempty"`                                                   // token identifier / ticker
	Amount            string   `protobuf:"bytes,4,opt,name=amount,proto3" json:"amount,omitempty"`                                                 // big.Int number of tokens to unlock
	Reason            string   `protobuf:"bytes,5,opt,name=reason,proto3" json:"reason,omitempty"`                                                 // reason for blocking
	Docs              [][]byte `protobuf:"bytes,6,rep,name=docs,proto3" json:"docs,omitempty"`                                                     // hashes of documents with justification (optional parameter)
	Payload           []byte   `protobuf:"bytes,7,opt,name=payload,proto3" json:"payload,omitempty"`                                               // additional information (optional parameter)
	CompleteOperation bool     `protobuf:"varint,8,opt,name=complete_operation,json=completeOperation,proto3" json:"complete_operation,omitempty"` // sign that it is completely unlocked
}

func (x *AllowedBalanceUnlocked) Reset() {
	*x = AllowedBalanceUnlocked{}
	if protoimpl.UnsafeEnabled {
		mi := &file_locks_proto_msgTypes[6]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *AllowedBalanceUnlocked) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*AllowedBalanceUnlocked) ProtoMessage() {}

func (x *AllowedBalanceUnlocked) ProtoReflect() protoreflect.Message {
	mi := &file_locks_proto_msgTypes[6]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use AllowedBalanceUnlocked.ProtoReflect.Descriptor instead.
func (*AllowedBalanceUnlocked) Descriptor() ([]byte, []int) {
	return file_locks_proto_rawDescGZIP(), []int{6}
}

func (x *AllowedBalanceUnlocked) GetId() string {
	if x != nil {
		return x.Id
	}
	return ""
}

func (x *AllowedBalanceUnlocked) GetAddress() string {
	if x != nil {
		return x.Address
	}
	return ""
}

func (x *AllowedBalanceUnlocked) GetToken() string {
	if x != nil {
		return x.Token
	}
	return ""
}

func (x *AllowedBalanceUnlocked) GetAmount() string {
	if x != nil {
		return x.Amount
	}
	return ""
}

func (x *AllowedBalanceUnlocked) GetReason() string {
	if x != nil {
		return x.Reason
	}
	return ""
}

func (x *AllowedBalanceUnlocked) GetDocs() [][]byte {
	if x != nil {
		return x.Docs
	}
	return nil
}

func (x *AllowedBalanceUnlocked) GetPayload() []byte {
	if x != nil {
		return x.Payload
	}
	return nil
}

func (x *AllowedBalanceUnlocked) GetCompleteOperation() bool {
	if x != nil {
		return x.CompleteOperation
	}
	return false
}

var File_locks_proto protoreflect.FileDescriptor

var file_locks_proto_rawDesc = []byte{
	0x0a, 0x0b, 0x6c, 0x6f, 0x63, 0x6b, 0x73, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x12, 0x05, 0x70,
	0x72, 0x6f, 0x74, 0x6f, 0x22, 0xb2, 0x01, 0x0a, 0x12, 0x42, 0x61, 0x6c, 0x61, 0x6e, 0x63, 0x65,
	0x4c, 0x6f, 0x63, 0x6b, 0x52, 0x65, 0x71, 0x75, 0x65, 0x73, 0x74, 0x12, 0x0e, 0x0a, 0x02, 0x69,
	0x64, 0x18, 0x01, 0x20, 0x01, 0x28, 0x09, 0x52, 0x02, 0x69, 0x64, 0x12, 0x18, 0x0a, 0x07, 0x61,
	0x64, 0x64, 0x72, 0x65, 0x73, 0x73, 0x18, 0x02, 0x20, 0x01, 0x28, 0x09, 0x52, 0x07, 0x61, 0x64,
	0x64, 0x72, 0x65, 0x73, 0x73, 0x12, 0x14, 0x0a, 0x05, 0x74, 0x6f, 0x6b, 0x65, 0x6e, 0x18, 0x03,
	0x20, 0x01, 0x28, 0x09, 0x52, 0x05, 0x74, 0x6f, 0x6b, 0x65, 0x6e, 0x12, 0x16, 0x0a, 0x06, 0x61,
	0x6d, 0x6f, 0x75, 0x6e, 0x74, 0x18, 0x04, 0x20, 0x01, 0x28, 0x09, 0x52, 0x06, 0x61, 0x6d, 0x6f,
	0x75, 0x6e, 0x74, 0x12, 0x16, 0x0a, 0x06, 0x72, 0x65, 0x61, 0x73, 0x6f, 0x6e, 0x18, 0x05, 0x20,
	0x01, 0x28, 0x09, 0x52, 0x06, 0x72, 0x65, 0x61, 0x73, 0x6f, 0x6e, 0x12, 0x12, 0x0a, 0x04, 0x64,
	0x6f, 0x63, 0x73, 0x18, 0x06, 0x20, 0x03, 0x28, 0x0c, 0x52, 0x04, 0x64, 0x6f, 0x63, 0x73, 0x12,
	0x18, 0x0a, 0x07, 0x70, 0x61, 0x79, 0x6c, 0x6f, 0x61, 0x64, 0x18, 0x07, 0x20, 0x01, 0x28, 0x0c,
	0x52, 0x07, 0x70, 0x61, 0x79, 0x6c, 0x6f, 0x61, 0x64, 0x22, 0xe0, 0x01, 0x0a, 0x10, 0x54, 0x6f,
	0x6b, 0x65, 0x6e, 0x42, 0x61, 0x6c, 0x61, 0x6e, 0x63, 0x65, 0x4c, 0x6f, 0x63, 0x6b, 0x12, 0x0e,
	0x0a, 0x02, 0x69, 0x64, 0x18, 0x01, 0x20, 0x01, 0x28, 0x09, 0x52, 0x02, 0x69, 0x64, 0x12, 0x18,
	0x0a, 0x07, 0x61, 0x64, 0x64, 0x72, 0x65, 0x73, 0x73, 0x18, 0x02, 0x20, 0x01, 0x28, 0x09, 0x52,
	0x07, 0x61, 0x64, 0x64, 0x72, 0x65, 0x73, 0x73, 0x12, 0x14, 0x0a, 0x05, 0x74, 0x6f, 0x6b, 0x65,
	0x6e, 0x18, 0x03, 0x20, 0x01, 0x28, 0x09, 0x52, 0x05, 0x74, 0x6f, 0x6b, 0x65, 0x6e, 0x12, 0x1f,
	0x0a, 0x0b, 0x69, 0x6e, 0x69, 0x74, 0x5f, 0x61, 0x6d, 0x6f, 0x75, 0x6e, 0x74, 0x18, 0x04, 0x20,
	0x01, 0x28, 0x09, 0x52, 0x0a, 0x69, 0x6e, 0x69, 0x74, 0x41, 0x6d, 0x6f, 0x75, 0x6e, 0x74, 0x12,
	0x25, 0x0a, 0x0e, 0x63, 0x75, 0x72, 0x72, 0x65, 0x6e, 0x74, 0x5f, 0x61, 0x6d, 0x6f, 0x75, 0x6e,
	0x74, 0x18, 0x05, 0x20, 0x01, 0x28, 0x09, 0x52, 0x0d, 0x63, 0x75, 0x72, 0x72, 0x65, 0x6e, 0x74,
	0x41, 0x6d, 0x6f, 0x75, 0x6e, 0x74, 0x12, 0x16, 0x0a, 0x06, 0x72, 0x65, 0x61, 0x73, 0x6f, 0x6e,
	0x18, 0x06, 0x20, 0x01, 0x28, 0x09, 0x52, 0x06, 0x72, 0x65, 0x61, 0x73, 0x6f, 0x6e, 0x12, 0x12,
	0x0a, 0x04, 0x64, 0x6f, 0x63, 0x73, 0x18, 0x07, 0x20, 0x03, 0x28, 0x0c, 0x52, 0x04, 0x64, 0x6f,
	0x63, 0x73, 0x12, 0x18, 0x0a, 0x07, 0x70, 0x61, 0x79, 0x6c, 0x6f, 0x61, 0x64, 0x18, 0x08, 0x20,
	0x01, 0x28, 0x0c, 0x52, 0x07, 0x70, 0x61, 0x79, 0x6c, 0x6f, 0x61, 0x64, 0x22, 0xe2, 0x01, 0x0a,
	0x12, 0x41, 0x6c, 0x6c, 0x6f, 0x77, 0x65, 0x64, 0x42, 0x61, 0x6c, 0x61, 0x6e, 0x63, 0x65, 0x4c,
	0x6f, 0x63, 0x6b, 0x12, 0x0e, 0x0a, 0x02, 0x69, 0x64, 0x18, 0x01, 0x20, 0x01, 0x28, 0x09, 0x52,
	0x02, 0x69, 0x64, 0x12, 0x18, 0x0a, 0x07, 0x61, 0x64, 0x64, 0x72, 0x65, 0x73, 0x73, 0x18, 0x02,
	0x20, 0x01, 0x28, 0x09, 0x52, 0x07, 0x61, 0x64, 0x64, 0x72, 0x65, 0x73, 0x73, 0x12, 0x14, 0x0a,
	0x05, 0x74, 0x6f, 0x6b, 0x65, 0x6e, 0x18, 0x03, 0x20, 0x01, 0x28, 0x09, 0x52, 0x05, 0x74, 0x6f,
	0x6b, 0x65, 0x6e, 0x12, 0x1f, 0x0a, 0x0b, 0x69, 0x6e, 0x69, 0x74, 0x5f, 0x61, 0x6d, 0x6f, 0x75,
	0x6e, 0x74, 0x18, 0x04, 0x20, 0x01, 0x28, 0x09, 0x52, 0x0a, 0x69, 0x6e, 0x69, 0x74, 0x41, 0x6d,
	0x6f, 0x75, 0x6e, 0x74, 0x12, 0x25, 0x0a, 0x0e, 0x63, 0x75, 0x72, 0x72, 0x65, 0x6e, 0x74, 0x5f,
	0x61, 0x6d, 0x6f, 0x75, 0x6e, 0x74, 0x18, 0x05, 0x20, 0x01, 0x28, 0x09, 0x52, 0x0d, 0x63, 0x75,
	0x72, 0x72, 0x65, 0x6e, 0x74, 0x41, 0x6d, 0x6f, 0x75, 0x6e, 0x74, 0x12, 0x16, 0x0a, 0x06, 0x72,
	0x65, 0x61, 0x73, 0x6f, 0x6e, 0x18, 0x06, 0x20, 0x01, 0x28, 0x09, 0x52, 0x06, 0x72, 0x65, 0x61,
	0x73, 0x6f, 0x6e, 0x12, 0x12, 0x0a, 0x04, 0x64, 0x6f, 0x63, 0x73, 0x18, 0x07, 0x20, 0x03, 0x28,
	0x0c, 0x52, 0x04, 0x64, 0x6f, 0x63, 0x73, 0x12, 0x18, 0x0a, 0x07, 0x70, 0x61, 0x79, 0x6c, 0x6f,
	0x61, 0x64, 0x18, 0x08, 0x20, 0x01, 0x28, 0x0c, 0x52, 0x07, 0x70, 0x61, 0x79, 0x6c, 0x6f, 0x61,
	0x64, 0x22, 0xb2, 0x01, 0x0a, 0x12, 0x54, 0x6f, 0x6b, 0x65, 0x6e, 0x42, 0x61, 0x6c, 0x61, 0x6e,
	0x63, 0x65, 0x4c, 0x6f, 0x63, 0x6b, 0x65, 0x64, 0x12, 0x0e, 0x0a, 0x02, 0x69, 0x64, 0x18, 0x01,
	0x20, 0x01, 0x28, 0x09, 0x52, 0x02, 0x69, 0x64, 0x12, 0x18, 0x0a, 0x07, 0x61, 0x64, 0x64, 0x72,
	0x65, 0x73, 0x73, 0x18, 0x02, 0x20, 0x01, 0x28, 0x09, 0x52, 0x07, 0x61, 0x64, 0x64, 0x72, 0x65,
	0x73, 0x73, 0x12, 0x14, 0x0a, 0x05, 0x74, 0x6f, 0x6b, 0x65, 0x6e, 0x18, 0x03, 0x20, 0x01, 0x28,
	0x09, 0x52, 0x05, 0x74, 0x6f, 0x6b, 0x65, 0x6e, 0x12, 0x16, 0x0a, 0x06, 0x61, 0x6d, 0x6f, 0x75,
	0x6e, 0x74, 0x18, 0x04, 0x20, 0x01, 0x28, 0x09, 0x52, 0x06, 0x61, 0x6d, 0x6f, 0x75, 0x6e, 0x74,
	0x12, 0x16, 0x0a, 0x06, 0x72, 0x65, 0x61, 0x73, 0x6f, 0x6e, 0x18, 0x05, 0x20, 0x01, 0x28, 0x09,
	0x52, 0x06, 0x72, 0x65, 0x61, 0x73, 0x6f, 0x6e, 0x12, 0x12, 0x0a, 0x04, 0x64, 0x6f, 0x63, 0x73,
	0x18, 0x06, 0x20, 0x03, 0x28, 0x0c, 0x52, 0x04, 0x64, 0x6f, 0x63, 0x73, 0x12, 0x18, 0x0a, 0x07,
	0x70, 0x61, 0x79, 0x6c, 0x6f, 0x61, 0x64, 0x18, 0x07, 0x20, 0x01, 0x28, 0x0c, 0x52, 0x07, 0x70,
	0x61, 0x79, 0x6c, 0x6f, 0x61, 0x64, 0x22, 0xe3, 0x01, 0x0a, 0x14, 0x54, 0x6f, 0x6b, 0x65, 0x6e,
	0x42, 0x61, 0x6c, 0x61, 0x6e, 0x63, 0x65, 0x55, 0x6e, 0x6c, 0x6f, 0x63, 0x6b, 0x65, 0x64, 0x12,
	0x0e, 0x0a, 0x02, 0x69, 0x64, 0x18, 0x01, 0x20, 0x01, 0x28, 0x09, 0x52, 0x02, 0x69, 0x64, 0x12,
	0x18, 0x0a, 0x07, 0x61, 0x64, 0x64, 0x72, 0x65, 0x73, 0x73, 0x18, 0x02, 0x20, 0x01, 0x28, 0x09,
	0x52, 0x07, 0x61, 0x64, 0x64, 0x72, 0x65, 0x73, 0x73, 0x12, 0x14, 0x0a, 0x05, 0x74, 0x6f, 0x6b,
	0x65, 0x6e, 0x18, 0x03, 0x20, 0x01, 0x28, 0x09, 0x52, 0x05, 0x74, 0x6f, 0x6b, 0x65, 0x6e, 0x12,
	0x16, 0x0a, 0x06, 0x61, 0x6d, 0x6f, 0x75, 0x6e, 0x74, 0x18, 0x04, 0x20, 0x01, 0x28, 0x09, 0x52,
	0x06, 0x61, 0x6d, 0x6f, 0x75, 0x6e, 0x74, 0x12, 0x16, 0x0a, 0x06, 0x72, 0x65, 0x61, 0x73, 0x6f,
	0x6e, 0x18, 0x05, 0x20, 0x01, 0x28, 0x09, 0x52, 0x06, 0x72, 0x65, 0x61, 0x73, 0x6f, 0x6e, 0x12,
	0x12, 0x0a, 0x04, 0x64, 0x6f, 0x63, 0x73, 0x18, 0x06, 0x20, 0x03, 0x28, 0x0c, 0x52, 0x04, 0x64,
	0x6f, 0x63, 0x73, 0x12, 0x18, 0x0a, 0x07, 0x70, 0x61, 0x79, 0x6c, 0x6f, 0x61, 0x64, 0x18, 0x07,
	0x20, 0x01, 0x28, 0x0c, 0x52, 0x07, 0x70, 0x61, 0x79, 0x6c, 0x6f, 0x61, 0x64, 0x12, 0x2d, 0x0a,
	0x12, 0x63, 0x6f, 0x6d, 0x70, 0x6c, 0x65, 0x74, 0x65, 0x5f, 0x6f, 0x70, 0x65, 0x72, 0x61, 0x74,
	0x69, 0x6f, 0x6e, 0x18, 0x08, 0x20, 0x01, 0x28, 0x08, 0x52, 0x11, 0x63, 0x6f, 0x6d, 0x70, 0x6c,
	0x65, 0x74, 0x65, 0x4f, 0x70, 0x65, 0x72, 0x61, 0x74, 0x69, 0x6f, 0x6e, 0x22, 0xb4, 0x01, 0x0a,
	0x14, 0x41, 0x6c, 0x6c, 0x6f, 0x77, 0x65, 0x64, 0x42, 0x61, 0x6c, 0x61, 0x6e, 0x63, 0x65, 0x4c,
	0x6f, 0x63, 0x6b, 0x65, 0x64, 0x12, 0x0e, 0x0a, 0x02, 0x69, 0x64, 0x18, 0x01, 0x20, 0x01, 0x28,
	0x09, 0x52, 0x02, 0x69, 0x64, 0x12, 0x18, 0x0a, 0x07, 0x61, 0x64, 0x64, 0x72, 0x65, 0x73, 0x73,
	0x18, 0x02, 0x20, 0x01, 0x28, 0x09, 0x52, 0x07, 0x61, 0x64, 0x64, 0x72, 0x65, 0x73, 0x73, 0x12,
	0x14, 0x0a, 0x05, 0x74, 0x6f, 0x6b, 0x65, 0x6e, 0x18, 0x03, 0x20, 0x01, 0x28, 0x09, 0x52, 0x05,
	0x74, 0x6f, 0x6b, 0x65, 0x6e, 0x12, 0x16, 0x0a, 0x06, 0x61, 0x6d, 0x6f, 0x75, 0x6e, 0x74, 0x18,
	0x04, 0x20, 0x01, 0x28, 0x09, 0x52, 0x06, 0x61, 0x6d, 0x6f, 0x75, 0x6e, 0x74, 0x12, 0x16, 0x0a,
	0x06, 0x72, 0x65, 0x61, 0x73, 0x6f, 0x6e, 0x18, 0x05, 0x20, 0x01, 0x28, 0x09, 0x52, 0x06, 0x72,
	0x65, 0x61, 0x73, 0x6f, 0x6e, 0x12, 0x12, 0x0a, 0x04, 0x64, 0x6f, 0x63, 0x73, 0x18, 0x06, 0x20,
	0x03, 0x28, 0x0c, 0x52, 0x04, 0x64, 0x6f, 0x63, 0x73, 0x12, 0x18, 0x0a, 0x07, 0x70, 0x61, 0x79,
	0x6c, 0x6f, 0x61, 0x64, 0x18, 0x07, 0x20, 0x01, 0x28, 0x0c, 0x52, 0x07, 0x70, 0x61, 0x79, 0x6c,
	0x6f, 0x61, 0x64, 0x22, 0xe5, 0x01, 0x0a, 0x16, 0x41, 0x6c, 0x6c, 0x6f, 0x77, 0x65, 0x64, 0x42,
	0x61, 0x6c, 0x61, 0x6e, 0x63, 0x65, 0x55, 0x6e, 0x6c, 0x6f, 0x63, 0x6b, 0x65, 0x64, 0x12, 0x0e,
	0x0a, 0x02, 0x69, 0x64, 0x18, 0x01, 0x20, 0x01, 0x28, 0x09, 0x52, 0x02, 0x69, 0x64, 0x12, 0x18,
	0x0a, 0x07, 0x61, 0x64, 0x64, 0x72, 0x65, 0x73, 0x73, 0x18, 0x02, 0x20, 0x01, 0x28, 0x09, 0x52,
	0x07, 0x61, 0x64, 0x64, 0x72, 0x65, 0x73, 0x73, 0x12, 0x14, 0x0a, 0x05, 0x74, 0x6f, 0x6b, 0x65,
	0x6e, 0x18, 0x03, 0x20, 0x01, 0x28, 0x09, 0x52, 0x05, 0x74, 0x6f, 0x6b, 0x65, 0x6e, 0x12, 0x16,
	0x0a, 0x06, 0x61, 0x6d, 0x6f, 0x75, 0x6e, 0x74, 0x18, 0x04, 0x20, 0x01, 0x28, 0x09, 0x52, 0x06,
	0x61, 0x6d, 0x6f, 0x75, 0x6e, 0x74, 0x12, 0x16, 0x0a, 0x06, 0x72, 0x65, 0x61, 0x73, 0x6f, 0x6e,
	0x18, 0x05, 0x20, 0x01, 0x28, 0x09, 0x52, 0x06, 0x72, 0x65, 0x61, 0x73, 0x6f, 0x6e, 0x12, 0x12,
	0x0a, 0x04, 0x64, 0x6f, 0x63, 0x73, 0x18, 0x06, 0x20, 0x03, 0x28, 0x0c, 0x52, 0x04, 0x64, 0x6f,
	0x63, 0x73, 0x12, 0x18, 0x0a, 0x07, 0x70, 0x61, 0x79, 0x6c, 0x6f, 0x61, 0x64, 0x18, 0x07, 0x20,
	0x01, 0x28, 0x0c, 0x52, 0x07, 0x70, 0x61, 0x79, 0x6c, 0x6f, 0x61, 0x64, 0x12, 0x2d, 0x0a, 0x12,
	0x63, 0x6f, 0x6d, 0x70, 0x6c, 0x65, 0x74, 0x65, 0x5f, 0x6f, 0x70, 0x65, 0x72, 0x61, 0x74, 0x69,
	0x6f, 0x6e, 0x18, 0x08, 0x20, 0x01, 0x28, 0x08, 0x52, 0x11, 0x63, 0x6f, 0x6d, 0x70, 0x6c, 0x65,
	0x74, 0x65, 0x4f, 0x70, 0x65, 0x72, 0x61, 0x74, 0x69, 0x6f, 0x6e, 0x42, 0x0a, 0x5a, 0x08, 0x2e,
	0x2f, 0x3b, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x62, 0x06, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x33,
}

var (
	file_locks_proto_rawDescOnce sync.Once
	file_locks_proto_rawDescData = file_locks_proto_rawDesc
)

func file_locks_proto_rawDescGZIP() []byte {
	file_locks_proto_rawDescOnce.Do(func() {
		file_locks_proto_rawDescData = protoimpl.X.CompressGZIP(file_locks_proto_rawDescData)
	})
	return file_locks_proto_rawDescData
}

var file_locks_proto_msgTypes = make([]protoimpl.MessageInfo, 7)
var file_locks_proto_goTypes = []interface{}{
	(*BalanceLockRequest)(nil),     // 0: proto.BalanceLockRequest
	(*TokenBalanceLock)(nil),       // 1: proto.TokenBalanceLock
	(*AllowedBalanceLock)(nil),     // 2: proto.AllowedBalanceLock
	(*TokenBalanceLocked)(nil),     // 3: proto.TokenBalanceLocked
	(*TokenBalanceUnlocked)(nil),   // 4: proto.TokenBalanceUnlocked
	(*AllowedBalanceLocked)(nil),   // 5: proto.AllowedBalanceLocked
	(*AllowedBalanceUnlocked)(nil), // 6: proto.AllowedBalanceUnlocked
}
var file_locks_proto_depIdxs = []int32{
	0, // [0:0] is the sub-list for method output_type
	0, // [0:0] is the sub-list for method input_type
	0, // [0:0] is the sub-list for extension type_name
	0, // [0:0] is the sub-list for extension extendee
	0, // [0:0] is the sub-list for field type_name
}

func init() { file_locks_proto_init() }
func file_locks_proto_init() {
	if File_locks_proto != nil {
		return
	}
	if !protoimpl.UnsafeEnabled {
		file_locks_proto_msgTypes[0].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*BalanceLockRequest); i {
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
		file_locks_proto_msgTypes[1].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*TokenBalanceLock); i {
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
		file_locks_proto_msgTypes[2].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*AllowedBalanceLock); i {
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
		file_locks_proto_msgTypes[3].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*TokenBalanceLocked); i {
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
		file_locks_proto_msgTypes[4].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*TokenBalanceUnlocked); i {
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
		file_locks_proto_msgTypes[5].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*AllowedBalanceLocked); i {
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
		file_locks_proto_msgTypes[6].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*AllowedBalanceUnlocked); i {
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
			RawDescriptor: file_locks_proto_rawDesc,
			NumEnums:      0,
			NumMessages:   7,
			NumExtensions: 0,
			NumServices:   0,
		},
		GoTypes:           file_locks_proto_goTypes,
		DependencyIndexes: file_locks_proto_depIdxs,
		MessageInfos:      file_locks_proto_msgTypes,
	}.Build()
	File_locks_proto = out.File
	file_locks_proto_rawDesc = nil
	file_locks_proto_goTypes = nil
	file_locks_proto_depIdxs = nil
}
