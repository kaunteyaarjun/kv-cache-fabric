// Code generated for kvblock proto messages. DO NOT EDIT.
package kvblock

import (
	protoreflect "google.golang.org/protobuf/reflect/protoreflect"
	protoimpl "google.golang.org/protobuf/runtime/protoimpl"
)

type AllocateBlockRequest struct {
	state         protoimpl.MessageState `protogen:"open.v1"`
	SequenceId    uint64                 `protobuf:"varint,1,opt,name=sequence_id,json=sequenceId,proto3" json:"sequence_id,omitempty"`
	NumTokens     uint32                 `protobuf:"varint,2,opt,name=num_tokens,json=numTokens,proto3" json:"num_tokens,omitempty"`
	InitialData   []byte                 `protobuf:"bytes,3,opt,name=initial_data,json=initialData,proto3" json:"initial_data,omitempty"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *AllocateBlockRequest) Reset() {
	*x = AllocateBlockRequest{}
}

func (x *AllocateBlockRequest) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*AllocateBlockRequest) ProtoMessage() {}

func (x *AllocateBlockRequest) ProtoReflect() protoreflect.Message {
	mi := &file_proto_kvblock_proto_msgTypes[0]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

func (x *AllocateBlockRequest) GetSequenceId() uint64 {
	if x != nil {
		return x.SequenceId
	}
	return 0
}

func (x *AllocateBlockRequest) GetNumTokens() uint32 {
	if x != nil {
		return x.NumTokens
	}
	return 0
}

func (x *AllocateBlockRequest) GetInitialData() []byte {
	if x != nil {
		return x.InitialData
	}
	return nil
}

type AllocateBlockResponse struct {
	state         protoimpl.MessageState `protogen:"open.v1"`
	BlockId       uint64                 `protobuf:"varint,1,opt,name=block_id,json=blockId,proto3" json:"block_id,omitempty"`
	Tier          string                 `protobuf:"bytes,2,opt,name=tier,proto3" json:"tier,omitempty"`
	CapacityBytes uint32                 `protobuf:"varint,3,opt,name=capacity_bytes,json=capacityBytes,proto3" json:"capacity_bytes,omitempty"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *AllocateBlockResponse) Reset() {
	*x = AllocateBlockResponse{}
}

func (x *AllocateBlockResponse) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*AllocateBlockResponse) ProtoMessage() {}

func (x *AllocateBlockResponse) ProtoReflect() protoreflect.Message {
	mi := &file_proto_kvblock_proto_msgTypes[1]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

func (x *AllocateBlockResponse) GetBlockId() uint64 {
	if x != nil {
		return x.BlockId
	}
	return 0
}

func (x *AllocateBlockResponse) GetTier() string {
	if x != nil {
		return x.Tier
	}
	return ""
}

func (x *AllocateBlockResponse) GetCapacityBytes() uint32 {
	if x != nil {
		return x.CapacityBytes
	}
	return 0
}

type TouchBlockRequest struct {
	state         protoimpl.MessageState `protogen:"open.v1"`
	BlockId       uint64                 `protobuf:"varint,1,opt,name=block_id,json=blockId,proto3" json:"block_id,omitempty"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *TouchBlockRequest) Reset() {
	*x = TouchBlockRequest{}
}

func (x *TouchBlockRequest) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*TouchBlockRequest) ProtoMessage() {}

func (x *TouchBlockRequest) ProtoReflect() protoreflect.Message {
	mi := &file_proto_kvblock_proto_msgTypes[2]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

func (x *TouchBlockRequest) GetBlockId() uint64 {
	if x != nil {
		return x.BlockId
	}
	return 0
}

type TouchBlockResponse struct {
	state         protoimpl.MessageState `protogen:"open.v1"`
	Success       bool                   `protobuf:"varint,1,opt,name=success,proto3" json:"success,omitempty"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *TouchBlockResponse) Reset() {
	*x = TouchBlockResponse{}
}

func (x *TouchBlockResponse) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*TouchBlockResponse) ProtoMessage() {}

func (x *TouchBlockResponse) ProtoReflect() protoreflect.Message {
	mi := &file_proto_kvblock_proto_msgTypes[3]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

func (x *TouchBlockResponse) GetSuccess() bool {
	if x != nil {
		return x.Success
	}
	return false
}

type GetBlockLocationRequest struct {
	state         protoimpl.MessageState `protogen:"open.v1"`
	BlockId       uint64                 `protobuf:"varint,1,opt,name=block_id,json=blockId,proto3" json:"block_id,omitempty"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *GetBlockLocationRequest) Reset() {
	*x = GetBlockLocationRequest{}
}

func (x *GetBlockLocationRequest) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*GetBlockLocationRequest) ProtoMessage() {}

func (x *GetBlockLocationRequest) ProtoReflect() protoreflect.Message {
	mi := &file_proto_kvblock_proto_msgTypes[4]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

func (x *GetBlockLocationRequest) GetBlockId() uint64 {
	if x != nil {
		return x.BlockId
	}
	return 0
}

type GetBlockLocationResponse struct {
	state         protoimpl.MessageState `protogen:"open.v1"`
	Tier          string                 `protobuf:"bytes,1,opt,name=tier,proto3" json:"tier,omitempty"`
	Exists        bool                   `protobuf:"varint,2,opt,name=exists,proto3" json:"exists,omitempty"`
	NumTokens     uint32                 `protobuf:"varint,3,opt,name=num_tokens,json=numTokens,proto3" json:"num_tokens,omitempty"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *GetBlockLocationResponse) Reset() {
	*x = GetBlockLocationResponse{}
}

func (x *GetBlockLocationResponse) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*GetBlockLocationResponse) ProtoMessage() {}

func (x *GetBlockLocationResponse) ProtoReflect() protoreflect.Message {
	mi := &file_proto_kvblock_proto_msgTypes[5]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

func (x *GetBlockLocationResponse) GetTier() string {
	if x != nil {
		return x.Tier
	}
	return ""
}

func (x *GetBlockLocationResponse) GetExists() bool {
	if x != nil {
		return x.Exists
	}
	return false
}

func (x *GetBlockLocationResponse) GetNumTokens() uint32 {
	if x != nil {
		return x.NumTokens
	}
	return 0
}

type EvictLRURequest struct {
	state         protoimpl.MessageState `protogen:"open.v1"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *EvictLRURequest) Reset() {
	*x = EvictLRURequest{}
}

func (x *EvictLRURequest) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*EvictLRURequest) ProtoMessage() {}

func (x *EvictLRURequest) ProtoReflect() protoreflect.Message {
	mi := &file_proto_kvblock_proto_msgTypes[6]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

type EvictLRUResponse struct {
	state         protoimpl.MessageState `protogen:"open.v1"`
	EvictedBlocks uint64                 `protobuf:"varint,1,opt,name=evicted_blocks,json=evictedBlocks,proto3" json:"evicted_blocks,omitempty"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *EvictLRUResponse) Reset() {
	*x = EvictLRUResponse{}
}

func (x *EvictLRUResponse) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*EvictLRUResponse) ProtoMessage() {}

func (x *EvictLRUResponse) ProtoReflect() protoreflect.Message {
	mi := &file_proto_kvblock_proto_msgTypes[7]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

func (x *EvictLRUResponse) GetEvictedBlocks() uint64 {
	if x != nil {
		return x.EvictedBlocks
	}
	return 0
}

type WriteBlockDataRequest struct {
	state         protoimpl.MessageState `protogen:"open.v1"`
	BlockId       uint64                 `protobuf:"varint,1,opt,name=block_id,json=blockId,proto3" json:"block_id,omitempty"`
	Offset        uint32                 `protobuf:"varint,2,opt,name=offset,proto3" json:"offset,omitempty"`
	Data          []byte                 `protobuf:"bytes,3,opt,name=data,proto3" json:"data,omitempty"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *WriteBlockDataRequest) Reset() {
	*x = WriteBlockDataRequest{}
}

func (x *WriteBlockDataRequest) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*WriteBlockDataRequest) ProtoMessage() {}

func (x *WriteBlockDataRequest) ProtoReflect() protoreflect.Message {
	mi := &file_proto_kvblock_proto_msgTypes[8]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

func (x *WriteBlockDataRequest) GetBlockId() uint64 {
	if x != nil {
		return x.BlockId
	}
	return 0
}

func (x *WriteBlockDataRequest) GetOffset() uint32 {
	if x != nil {
		return x.Offset
	}
	return 0
}

func (x *WriteBlockDataRequest) GetData() []byte {
	if x != nil {
		return x.Data
	}
	return nil
}

type WriteBlockDataResponse struct {
	state         protoimpl.MessageState `protogen:"open.v1"`
	Success       bool                   `protobuf:"varint,1,opt,name=success,proto3" json:"success,omitempty"`
	BytesWritten  uint32                 `protobuf:"varint,2,opt,name=bytes_written,json=bytesWritten,proto3" json:"bytes_written,omitempty"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *WriteBlockDataResponse) Reset() {
	*x = WriteBlockDataResponse{}
}

func (x *WriteBlockDataResponse) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*WriteBlockDataResponse) ProtoMessage() {}

func (x *WriteBlockDataResponse) ProtoReflect() protoreflect.Message {
	mi := &file_proto_kvblock_proto_msgTypes[9]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

func (x *WriteBlockDataResponse) GetSuccess() bool {
	if x != nil {
		return x.Success
	}
	return false
}

func (x *WriteBlockDataResponse) GetBytesWritten() uint32 {
	if x != nil {
		return x.BytesWritten
	}
	return 0
}

type ReadBlockDataRequest struct {
	state         protoimpl.MessageState `protogen:"open.v1"`
	BlockId       uint64                 `protobuf:"varint,1,opt,name=block_id,json=blockId,proto3" json:"block_id,omitempty"`
	Offset        uint32                 `protobuf:"varint,2,opt,name=offset,proto3" json:"offset,omitempty"`
	Length        uint32                 `protobuf:"varint,3,opt,name=length,proto3" json:"length,omitempty"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *ReadBlockDataRequest) Reset() {
	*x = ReadBlockDataRequest{}
}

func (x *ReadBlockDataRequest) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*ReadBlockDataRequest) ProtoMessage() {}

func (x *ReadBlockDataRequest) ProtoReflect() protoreflect.Message {
	mi := &file_proto_kvblock_proto_msgTypes[10]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

func (x *ReadBlockDataRequest) GetBlockId() uint64 {
	if x != nil {
		return x.BlockId
	}
	return 0
}

func (x *ReadBlockDataRequest) GetOffset() uint32 {
	if x != nil {
		return x.Offset
	}
	return 0
}

func (x *ReadBlockDataRequest) GetLength() uint32 {
	if x != nil {
		return x.Length
	}
	return 0
}

type ReadBlockDataResponse struct {
	state         protoimpl.MessageState `protogen:"open.v1"`
	Data          []byte                 `protobuf:"bytes,1,opt,name=data,proto3" json:"data,omitempty"`
	Tier          string                 `protobuf:"bytes,2,opt,name=tier,proto3" json:"tier,omitempty"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *ReadBlockDataResponse) Reset() {
	*x = ReadBlockDataResponse{}
}

func (x *ReadBlockDataResponse) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*ReadBlockDataResponse) ProtoMessage() {}

func (x *ReadBlockDataResponse) ProtoReflect() protoreflect.Message {
	mi := &file_proto_kvblock_proto_msgTypes[11]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

func (x *ReadBlockDataResponse) GetData() []byte {
	if x != nil {
		return x.Data
	}
	return nil
}

func (x *ReadBlockDataResponse) GetTier() string {
	if x != nil {
		return x.Tier
	}
	return ""
}

var file_proto_kvblock_proto_msgTypes = make([]protoimpl.MessageInfo, 12)

func init() {
	file_proto_kvblock_proto_init()
}

func file_proto_kvblock_proto_init() {
	file_proto_kvblock_proto_msgTypes[0].Exporter = func(v any, i int) any {
		switch v := v.(*AllocateBlockRequest); i {
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
	file_proto_kvblock_proto_msgTypes[1].Exporter = func(v any, i int) any {
		switch v := v.(*AllocateBlockResponse); i {
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
	file_proto_kvblock_proto_msgTypes[2].Exporter = func(v any, i int) any {
		switch v := v.(*TouchBlockRequest); i {
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
	file_proto_kvblock_proto_msgTypes[3].Exporter = func(v any, i int) any {
		switch v := v.(*TouchBlockResponse); i {
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
	file_proto_kvblock_proto_msgTypes[4].Exporter = func(v any, i int) any {
		switch v := v.(*GetBlockLocationRequest); i {
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
	file_proto_kvblock_proto_msgTypes[5].Exporter = func(v any, i int) any {
		switch v := v.(*GetBlockLocationResponse); i {
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
	file_proto_kvblock_proto_msgTypes[6].Exporter = func(v any, i int) any {
		switch v := v.(*EvictLRURequest); i {
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
	file_proto_kvblock_proto_msgTypes[7].Exporter = func(v any, i int) any {
		switch v := v.(*EvictLRUResponse); i {
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
	file_proto_kvblock_proto_msgTypes[8].Exporter = func(v any, i int) any {
		switch v := v.(*WriteBlockDataRequest); i {
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
	file_proto_kvblock_proto_msgTypes[9].Exporter = func(v any, i int) any {
		switch v := v.(*WriteBlockDataResponse); i {
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
	file_proto_kvblock_proto_msgTypes[10].Exporter = func(v any, i int) any {
		switch v := v.(*ReadBlockDataRequest); i {
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
	file_proto_kvblock_proto_msgTypes[11].Exporter = func(v any, i int) any {
		switch v := v.(*ReadBlockDataResponse); i {
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
