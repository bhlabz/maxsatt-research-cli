# -*- coding: utf-8 -*-
# Generated by the protocol buffer compiler.  DO NOT EDIT!
# source: clear_and_smooth.proto
# Protobuf Python Version: 4.25.1
"""Generated protocol buffer code."""
from google.protobuf import descriptor as _descriptor
from google.protobuf import descriptor_pool as _descriptor_pool
from google.protobuf import symbol_database as _symbol_database
from google.protobuf.internal import builder as _builder
# @@protoc_insertion_point(imports)

_sym_db = _symbol_database.Default()




DESCRIPTOR = _descriptor_pool.Default().AddSerializedFile(b'\n\x16\x63lear_and_smooth.proto\"\x81\x01\n\x15\x43learAndSmoothRequest\x12.\n\x04\x64\x61ta\x18\x01 \x03(\x0b\x32 .ClearAndSmoothRequest.DataEntry\x1a\x38\n\tDataEntry\x12\x0b\n\x03key\x18\x01 \x01(\t\x12\x1a\n\x05value\x18\x02 \x01(\x0b\x32\x0b.DoubleList:\x02\x38\x01\"\x9c\x01\n\x16\x43learAndSmoothResponse\x12@\n\rsmoothed_data\x18\x01 \x03(\x0b\x32).ClearAndSmoothResponse.SmoothedDataEntry\x1a@\n\x11SmoothedDataEntry\x12\x0b\n\x03key\x18\x01 \x01(\t\x12\x1a\n\x05value\x18\x02 \x01(\x0b\x32\x0b.DoubleList:\x02\x38\x01\"\x1c\n\nDoubleList\x12\x0e\n\x06values\x18\x01 \x03(\x01\x32Z\n\x15\x43learAndSmoothService\x12\x41\n\x0e\x43learAndSmooth\x12\x16.ClearAndSmoothRequest\x1a\x17.ClearAndSmoothResponseB\x0cZ\n/protobufsb\x06proto3')

_globals = globals()
_builder.BuildMessageAndEnumDescriptors(DESCRIPTOR, _globals)
_builder.BuildTopDescriptorsAndMessages(DESCRIPTOR, 'clear_and_smooth_pb2', _globals)
if _descriptor._USE_C_DESCRIPTORS == False:
  _globals['DESCRIPTOR']._options = None
  _globals['DESCRIPTOR']._serialized_options = b'Z\n/protobufs'
  _globals['_CLEARANDSMOOTHREQUEST_DATAENTRY']._options = None
  _globals['_CLEARANDSMOOTHREQUEST_DATAENTRY']._serialized_options = b'8\001'
  _globals['_CLEARANDSMOOTHRESPONSE_SMOOTHEDDATAENTRY']._options = None
  _globals['_CLEARANDSMOOTHRESPONSE_SMOOTHEDDATAENTRY']._serialized_options = b'8\001'
  _globals['_CLEARANDSMOOTHREQUEST']._serialized_start=27
  _globals['_CLEARANDSMOOTHREQUEST']._serialized_end=156
  _globals['_CLEARANDSMOOTHREQUEST_DATAENTRY']._serialized_start=100
  _globals['_CLEARANDSMOOTHREQUEST_DATAENTRY']._serialized_end=156
  _globals['_CLEARANDSMOOTHRESPONSE']._serialized_start=159
  _globals['_CLEARANDSMOOTHRESPONSE']._serialized_end=315
  _globals['_CLEARANDSMOOTHRESPONSE_SMOOTHEDDATAENTRY']._serialized_start=251
  _globals['_CLEARANDSMOOTHRESPONSE_SMOOTHEDDATAENTRY']._serialized_end=315
  _globals['_DOUBLELIST']._serialized_start=317
  _globals['_DOUBLELIST']._serialized_end=345
  _globals['_CLEARANDSMOOTHSERVICE']._serialized_start=347
  _globals['_CLEARANDSMOOTHSERVICE']._serialized_end=437
# @@protoc_insertion_point(module_scope)
