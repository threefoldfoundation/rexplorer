# Generated by the protocol buffer compiler.  DO NOT EDIT!
# source: types.proto

import sys
_b=sys.version_info[0]<3 and (lambda x:x) or (lambda x:x.encode('latin1'))
from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from google.protobuf import reflection as _reflection
from google.protobuf import symbol_database as _symbol_database
# @@protoc_insertion_point(imports)

_sym_db = _symbol_database.Default()




DESCRIPTOR = _descriptor.FileDescriptor(
  name='types.proto',
  package='types',
  syntax='proto2',
  serialized_options=None,
  serialized_pb=_b('\n\x0btypes.proto\x12\x05types\"\xd9\x03\n\x0ePBNetworkStats\x12\x11\n\ttimestamp\x18\x01 \x02(\x04\x12\x13\n\x0b\x62lockheight\x18\x02 \x02(\x04\x12\x10\n\x08tx_count\x18\x03 \x02(\x04\x12\x1e\n\x16\x63oin_creation_tx_count\x18\x04 \x02(\x04\x12!\n\x19\x63oin_creator_def_tx_count\x18\x05 \x02(\x04\x12\x16\n\x0evalue_tx_count\x18\x06 \x02(\x04\x12\x19\n\x11\x63oin_output_count\x18\x07 \x02(\x04\x12 \n\x18locked_coin_output_count\x18\x08 \x02(\x04\x12\x18\n\x10\x63oin_input_count\x18\t \x02(\x04\x12\x1a\n\x12miner_payout_count\x18\n \x02(\x04\x12\x14\n\x0ctx_fee_count\x18\x0b \x02(\x04\x12\x15\n\rminer_payouts\x18\x0c \x02(\x0c\x12\x0f\n\x07tx_fees\x18\r \x02(\x0c\x12\r\n\x05\x63oins\x18\x0e \x02(\x0c\x12\x14\n\x0clocked_coins\x18\x0f \x02(\x0c\x12\x30\n(three_bot_registration_transaction_count\x18\x10 \x01(\x04\x12*\n\"three_bot_update_transaction_count\x18\x11 \x01(\x04\"\xcd\x01\n\x08PBWallet\x12\x38\n\x10\x62\x61lance_unlocked\x18\x01 \x01(\x0b\x32\x1e.types.PBWalletUnlockedBalance\x12\x34\n\x0e\x62\x61lance_locked\x18\x02 \x01(\x0b\x32\x1c.types.PBWalletLockedBalance\x12\x1b\n\x13multisign_addresses\x18\x03 \x03(\x0c\x12\x34\n\x0emultisign_data\x18\x04 \x01(\x0b\x32\x1c.types.PBWalletMultiSignData\"\xb5\x01\n\x17PBWalletUnlockedBalance\x12\r\n\x05total\x18\x01 \x02(\x0c\x12<\n\x07outputs\x18\x02 \x03(\x0b\x32+.types.PBWalletUnlockedBalance.OutputsEntry\x1aM\n\x0cOutputsEntry\x12\x0b\n\x03key\x18\x01 \x01(\t\x12,\n\x05value\x18\x02 \x01(\x0b\x32\x1d.types.PBWalletUnlockedOutput:\x02\x38\x01\"=\n\x16PBWalletUnlockedOutput\x12\x0e\n\x06\x61mount\x18\x01 \x02(\x0c\x12\x13\n\x0b\x64\x65scription\x18\x02 \x01(\t\"\xaf\x01\n\x15PBWalletLockedBalance\x12\r\n\x05total\x18\x01 \x02(\x0c\x12:\n\x07outputs\x18\x02 \x03(\x0b\x32).types.PBWalletLockedBalance.OutputsEntry\x1aK\n\x0cOutputsEntry\x12\x0b\n\x03key\x18\x01 \x01(\t\x12*\n\x05value\x18\x02 \x01(\x0b\x32\x1b.types.PBWalletLockedOutput:\x02\x38\x01\"Q\n\x14PBWalletLockedOutput\x12\x0e\n\x06\x61mount\x18\x01 \x02(\x0c\x12\x14\n\x0clocked_until\x18\x02 \x02(\x04\x12\x13\n\x0b\x64\x65scription\x18\x03 \x01(\t\"D\n\x15PBWalletMultiSignData\x12\x1b\n\x13signatures_required\x18\x01 \x02(\x04\x12\x0e\n\x06owners\x18\x02 \x03(\x0c\"u\n\x10PBThreeBotRecord\x12\n\n\x02id\x18\x01 \x02(\r\x12\x19\n\x11network_addresses\x18\x02 \x02(\x0c\x12\r\n\x05names\x18\x03 \x02(\x0c\x12\x17\n\x0f\x65xpiration_time\x18\x04 \x02(\x0c\x12\x12\n\npublic_key\x18\x05 \x02(\x0c')
)




_PBNETWORKSTATS = _descriptor.Descriptor(
  name='PBNetworkStats',
  full_name='types.PBNetworkStats',
  filename=None,
  file=DESCRIPTOR,
  containing_type=None,
  fields=[
    _descriptor.FieldDescriptor(
      name='timestamp', full_name='types.PBNetworkStats.timestamp', index=0,
      number=1, type=4, cpp_type=4, label=2,
      has_default_value=False, default_value=0,
      message_type=None, enum_type=None, containing_type=None,
      is_extension=False, extension_scope=None,
      serialized_options=None, file=DESCRIPTOR),
    _descriptor.FieldDescriptor(
      name='blockheight', full_name='types.PBNetworkStats.blockheight', index=1,
      number=2, type=4, cpp_type=4, label=2,
      has_default_value=False, default_value=0,
      message_type=None, enum_type=None, containing_type=None,
      is_extension=False, extension_scope=None,
      serialized_options=None, file=DESCRIPTOR),
    _descriptor.FieldDescriptor(
      name='tx_count', full_name='types.PBNetworkStats.tx_count', index=2,
      number=3, type=4, cpp_type=4, label=2,
      has_default_value=False, default_value=0,
      message_type=None, enum_type=None, containing_type=None,
      is_extension=False, extension_scope=None,
      serialized_options=None, file=DESCRIPTOR),
    _descriptor.FieldDescriptor(
      name='coin_creation_tx_count', full_name='types.PBNetworkStats.coin_creation_tx_count', index=3,
      number=4, type=4, cpp_type=4, label=2,
      has_default_value=False, default_value=0,
      message_type=None, enum_type=None, containing_type=None,
      is_extension=False, extension_scope=None,
      serialized_options=None, file=DESCRIPTOR),
    _descriptor.FieldDescriptor(
      name='coin_creator_def_tx_count', full_name='types.PBNetworkStats.coin_creator_def_tx_count', index=4,
      number=5, type=4, cpp_type=4, label=2,
      has_default_value=False, default_value=0,
      message_type=None, enum_type=None, containing_type=None,
      is_extension=False, extension_scope=None,
      serialized_options=None, file=DESCRIPTOR),
    _descriptor.FieldDescriptor(
      name='value_tx_count', full_name='types.PBNetworkStats.value_tx_count', index=5,
      number=6, type=4, cpp_type=4, label=2,
      has_default_value=False, default_value=0,
      message_type=None, enum_type=None, containing_type=None,
      is_extension=False, extension_scope=None,
      serialized_options=None, file=DESCRIPTOR),
    _descriptor.FieldDescriptor(
      name='coin_output_count', full_name='types.PBNetworkStats.coin_output_count', index=6,
      number=7, type=4, cpp_type=4, label=2,
      has_default_value=False, default_value=0,
      message_type=None, enum_type=None, containing_type=None,
      is_extension=False, extension_scope=None,
      serialized_options=None, file=DESCRIPTOR),
    _descriptor.FieldDescriptor(
      name='locked_coin_output_count', full_name='types.PBNetworkStats.locked_coin_output_count', index=7,
      number=8, type=4, cpp_type=4, label=2,
      has_default_value=False, default_value=0,
      message_type=None, enum_type=None, containing_type=None,
      is_extension=False, extension_scope=None,
      serialized_options=None, file=DESCRIPTOR),
    _descriptor.FieldDescriptor(
      name='coin_input_count', full_name='types.PBNetworkStats.coin_input_count', index=8,
      number=9, type=4, cpp_type=4, label=2,
      has_default_value=False, default_value=0,
      message_type=None, enum_type=None, containing_type=None,
      is_extension=False, extension_scope=None,
      serialized_options=None, file=DESCRIPTOR),
    _descriptor.FieldDescriptor(
      name='miner_payout_count', full_name='types.PBNetworkStats.miner_payout_count', index=9,
      number=10, type=4, cpp_type=4, label=2,
      has_default_value=False, default_value=0,
      message_type=None, enum_type=None, containing_type=None,
      is_extension=False, extension_scope=None,
      serialized_options=None, file=DESCRIPTOR),
    _descriptor.FieldDescriptor(
      name='tx_fee_count', full_name='types.PBNetworkStats.tx_fee_count', index=10,
      number=11, type=4, cpp_type=4, label=2,
      has_default_value=False, default_value=0,
      message_type=None, enum_type=None, containing_type=None,
      is_extension=False, extension_scope=None,
      serialized_options=None, file=DESCRIPTOR),
    _descriptor.FieldDescriptor(
      name='miner_payouts', full_name='types.PBNetworkStats.miner_payouts', index=11,
      number=12, type=12, cpp_type=9, label=2,
      has_default_value=False, default_value=_b(""),
      message_type=None, enum_type=None, containing_type=None,
      is_extension=False, extension_scope=None,
      serialized_options=None, file=DESCRIPTOR),
    _descriptor.FieldDescriptor(
      name='tx_fees', full_name='types.PBNetworkStats.tx_fees', index=12,
      number=13, type=12, cpp_type=9, label=2,
      has_default_value=False, default_value=_b(""),
      message_type=None, enum_type=None, containing_type=None,
      is_extension=False, extension_scope=None,
      serialized_options=None, file=DESCRIPTOR),
    _descriptor.FieldDescriptor(
      name='coins', full_name='types.PBNetworkStats.coins', index=13,
      number=14, type=12, cpp_type=9, label=2,
      has_default_value=False, default_value=_b(""),
      message_type=None, enum_type=None, containing_type=None,
      is_extension=False, extension_scope=None,
      serialized_options=None, file=DESCRIPTOR),
    _descriptor.FieldDescriptor(
      name='locked_coins', full_name='types.PBNetworkStats.locked_coins', index=14,
      number=15, type=12, cpp_type=9, label=2,
      has_default_value=False, default_value=_b(""),
      message_type=None, enum_type=None, containing_type=None,
      is_extension=False, extension_scope=None,
      serialized_options=None, file=DESCRIPTOR),
    _descriptor.FieldDescriptor(
      name='three_bot_registration_transaction_count', full_name='types.PBNetworkStats.three_bot_registration_transaction_count', index=15,
      number=16, type=4, cpp_type=4, label=1,
      has_default_value=False, default_value=0,
      message_type=None, enum_type=None, containing_type=None,
      is_extension=False, extension_scope=None,
      serialized_options=None, file=DESCRIPTOR),
    _descriptor.FieldDescriptor(
      name='three_bot_update_transaction_count', full_name='types.PBNetworkStats.three_bot_update_transaction_count', index=16,
      number=17, type=4, cpp_type=4, label=1,
      has_default_value=False, default_value=0,
      message_type=None, enum_type=None, containing_type=None,
      is_extension=False, extension_scope=None,
      serialized_options=None, file=DESCRIPTOR),
  ],
  extensions=[
  ],
  nested_types=[],
  enum_types=[
  ],
  serialized_options=None,
  is_extendable=False,
  syntax='proto2',
  extension_ranges=[],
  oneofs=[
  ],
  serialized_start=23,
  serialized_end=496,
)


_PBWALLET = _descriptor.Descriptor(
  name='PBWallet',
  full_name='types.PBWallet',
  filename=None,
  file=DESCRIPTOR,
  containing_type=None,
  fields=[
    _descriptor.FieldDescriptor(
      name='balance_unlocked', full_name='types.PBWallet.balance_unlocked', index=0,
      number=1, type=11, cpp_type=10, label=1,
      has_default_value=False, default_value=None,
      message_type=None, enum_type=None, containing_type=None,
      is_extension=False, extension_scope=None,
      serialized_options=None, file=DESCRIPTOR),
    _descriptor.FieldDescriptor(
      name='balance_locked', full_name='types.PBWallet.balance_locked', index=1,
      number=2, type=11, cpp_type=10, label=1,
      has_default_value=False, default_value=None,
      message_type=None, enum_type=None, containing_type=None,
      is_extension=False, extension_scope=None,
      serialized_options=None, file=DESCRIPTOR),
    _descriptor.FieldDescriptor(
      name='multisign_addresses', full_name='types.PBWallet.multisign_addresses', index=2,
      number=3, type=12, cpp_type=9, label=3,
      has_default_value=False, default_value=[],
      message_type=None, enum_type=None, containing_type=None,
      is_extension=False, extension_scope=None,
      serialized_options=None, file=DESCRIPTOR),
    _descriptor.FieldDescriptor(
      name='multisign_data', full_name='types.PBWallet.multisign_data', index=3,
      number=4, type=11, cpp_type=10, label=1,
      has_default_value=False, default_value=None,
      message_type=None, enum_type=None, containing_type=None,
      is_extension=False, extension_scope=None,
      serialized_options=None, file=DESCRIPTOR),
  ],
  extensions=[
  ],
  nested_types=[],
  enum_types=[
  ],
  serialized_options=None,
  is_extendable=False,
  syntax='proto2',
  extension_ranges=[],
  oneofs=[
  ],
  serialized_start=499,
  serialized_end=704,
)


_PBWALLETUNLOCKEDBALANCE_OUTPUTSENTRY = _descriptor.Descriptor(
  name='OutputsEntry',
  full_name='types.PBWalletUnlockedBalance.OutputsEntry',
  filename=None,
  file=DESCRIPTOR,
  containing_type=None,
  fields=[
    _descriptor.FieldDescriptor(
      name='key', full_name='types.PBWalletUnlockedBalance.OutputsEntry.key', index=0,
      number=1, type=9, cpp_type=9, label=1,
      has_default_value=False, default_value=_b("").decode('utf-8'),
      message_type=None, enum_type=None, containing_type=None,
      is_extension=False, extension_scope=None,
      serialized_options=None, file=DESCRIPTOR),
    _descriptor.FieldDescriptor(
      name='value', full_name='types.PBWalletUnlockedBalance.OutputsEntry.value', index=1,
      number=2, type=11, cpp_type=10, label=1,
      has_default_value=False, default_value=None,
      message_type=None, enum_type=None, containing_type=None,
      is_extension=False, extension_scope=None,
      serialized_options=None, file=DESCRIPTOR),
  ],
  extensions=[
  ],
  nested_types=[],
  enum_types=[
  ],
  serialized_options=_b('8\001'),
  is_extendable=False,
  syntax='proto2',
  extension_ranges=[],
  oneofs=[
  ],
  serialized_start=811,
  serialized_end=888,
)

_PBWALLETUNLOCKEDBALANCE = _descriptor.Descriptor(
  name='PBWalletUnlockedBalance',
  full_name='types.PBWalletUnlockedBalance',
  filename=None,
  file=DESCRIPTOR,
  containing_type=None,
  fields=[
    _descriptor.FieldDescriptor(
      name='total', full_name='types.PBWalletUnlockedBalance.total', index=0,
      number=1, type=12, cpp_type=9, label=2,
      has_default_value=False, default_value=_b(""),
      message_type=None, enum_type=None, containing_type=None,
      is_extension=False, extension_scope=None,
      serialized_options=None, file=DESCRIPTOR),
    _descriptor.FieldDescriptor(
      name='outputs', full_name='types.PBWalletUnlockedBalance.outputs', index=1,
      number=2, type=11, cpp_type=10, label=3,
      has_default_value=False, default_value=[],
      message_type=None, enum_type=None, containing_type=None,
      is_extension=False, extension_scope=None,
      serialized_options=None, file=DESCRIPTOR),
  ],
  extensions=[
  ],
  nested_types=[_PBWALLETUNLOCKEDBALANCE_OUTPUTSENTRY, ],
  enum_types=[
  ],
  serialized_options=None,
  is_extendable=False,
  syntax='proto2',
  extension_ranges=[],
  oneofs=[
  ],
  serialized_start=707,
  serialized_end=888,
)


_PBWALLETUNLOCKEDOUTPUT = _descriptor.Descriptor(
  name='PBWalletUnlockedOutput',
  full_name='types.PBWalletUnlockedOutput',
  filename=None,
  file=DESCRIPTOR,
  containing_type=None,
  fields=[
    _descriptor.FieldDescriptor(
      name='amount', full_name='types.PBWalletUnlockedOutput.amount', index=0,
      number=1, type=12, cpp_type=9, label=2,
      has_default_value=False, default_value=_b(""),
      message_type=None, enum_type=None, containing_type=None,
      is_extension=False, extension_scope=None,
      serialized_options=None, file=DESCRIPTOR),
    _descriptor.FieldDescriptor(
      name='description', full_name='types.PBWalletUnlockedOutput.description', index=1,
      number=2, type=9, cpp_type=9, label=1,
      has_default_value=False, default_value=_b("").decode('utf-8'),
      message_type=None, enum_type=None, containing_type=None,
      is_extension=False, extension_scope=None,
      serialized_options=None, file=DESCRIPTOR),
  ],
  extensions=[
  ],
  nested_types=[],
  enum_types=[
  ],
  serialized_options=None,
  is_extendable=False,
  syntax='proto2',
  extension_ranges=[],
  oneofs=[
  ],
  serialized_start=890,
  serialized_end=951,
)


_PBWALLETLOCKEDBALANCE_OUTPUTSENTRY = _descriptor.Descriptor(
  name='OutputsEntry',
  full_name='types.PBWalletLockedBalance.OutputsEntry',
  filename=None,
  file=DESCRIPTOR,
  containing_type=None,
  fields=[
    _descriptor.FieldDescriptor(
      name='key', full_name='types.PBWalletLockedBalance.OutputsEntry.key', index=0,
      number=1, type=9, cpp_type=9, label=1,
      has_default_value=False, default_value=_b("").decode('utf-8'),
      message_type=None, enum_type=None, containing_type=None,
      is_extension=False, extension_scope=None,
      serialized_options=None, file=DESCRIPTOR),
    _descriptor.FieldDescriptor(
      name='value', full_name='types.PBWalletLockedBalance.OutputsEntry.value', index=1,
      number=2, type=11, cpp_type=10, label=1,
      has_default_value=False, default_value=None,
      message_type=None, enum_type=None, containing_type=None,
      is_extension=False, extension_scope=None,
      serialized_options=None, file=DESCRIPTOR),
  ],
  extensions=[
  ],
  nested_types=[],
  enum_types=[
  ],
  serialized_options=_b('8\001'),
  is_extendable=False,
  syntax='proto2',
  extension_ranges=[],
  oneofs=[
  ],
  serialized_start=1054,
  serialized_end=1129,
)

_PBWALLETLOCKEDBALANCE = _descriptor.Descriptor(
  name='PBWalletLockedBalance',
  full_name='types.PBWalletLockedBalance',
  filename=None,
  file=DESCRIPTOR,
  containing_type=None,
  fields=[
    _descriptor.FieldDescriptor(
      name='total', full_name='types.PBWalletLockedBalance.total', index=0,
      number=1, type=12, cpp_type=9, label=2,
      has_default_value=False, default_value=_b(""),
      message_type=None, enum_type=None, containing_type=None,
      is_extension=False, extension_scope=None,
      serialized_options=None, file=DESCRIPTOR),
    _descriptor.FieldDescriptor(
      name='outputs', full_name='types.PBWalletLockedBalance.outputs', index=1,
      number=2, type=11, cpp_type=10, label=3,
      has_default_value=False, default_value=[],
      message_type=None, enum_type=None, containing_type=None,
      is_extension=False, extension_scope=None,
      serialized_options=None, file=DESCRIPTOR),
  ],
  extensions=[
  ],
  nested_types=[_PBWALLETLOCKEDBALANCE_OUTPUTSENTRY, ],
  enum_types=[
  ],
  serialized_options=None,
  is_extendable=False,
  syntax='proto2',
  extension_ranges=[],
  oneofs=[
  ],
  serialized_start=954,
  serialized_end=1129,
)


_PBWALLETLOCKEDOUTPUT = _descriptor.Descriptor(
  name='PBWalletLockedOutput',
  full_name='types.PBWalletLockedOutput',
  filename=None,
  file=DESCRIPTOR,
  containing_type=None,
  fields=[
    _descriptor.FieldDescriptor(
      name='amount', full_name='types.PBWalletLockedOutput.amount', index=0,
      number=1, type=12, cpp_type=9, label=2,
      has_default_value=False, default_value=_b(""),
      message_type=None, enum_type=None, containing_type=None,
      is_extension=False, extension_scope=None,
      serialized_options=None, file=DESCRIPTOR),
    _descriptor.FieldDescriptor(
      name='locked_until', full_name='types.PBWalletLockedOutput.locked_until', index=1,
      number=2, type=4, cpp_type=4, label=2,
      has_default_value=False, default_value=0,
      message_type=None, enum_type=None, containing_type=None,
      is_extension=False, extension_scope=None,
      serialized_options=None, file=DESCRIPTOR),
    _descriptor.FieldDescriptor(
      name='description', full_name='types.PBWalletLockedOutput.description', index=2,
      number=3, type=9, cpp_type=9, label=1,
      has_default_value=False, default_value=_b("").decode('utf-8'),
      message_type=None, enum_type=None, containing_type=None,
      is_extension=False, extension_scope=None,
      serialized_options=None, file=DESCRIPTOR),
  ],
  extensions=[
  ],
  nested_types=[],
  enum_types=[
  ],
  serialized_options=None,
  is_extendable=False,
  syntax='proto2',
  extension_ranges=[],
  oneofs=[
  ],
  serialized_start=1131,
  serialized_end=1212,
)


_PBWALLETMULTISIGNDATA = _descriptor.Descriptor(
  name='PBWalletMultiSignData',
  full_name='types.PBWalletMultiSignData',
  filename=None,
  file=DESCRIPTOR,
  containing_type=None,
  fields=[
    _descriptor.FieldDescriptor(
      name='signatures_required', full_name='types.PBWalletMultiSignData.signatures_required', index=0,
      number=1, type=4, cpp_type=4, label=2,
      has_default_value=False, default_value=0,
      message_type=None, enum_type=None, containing_type=None,
      is_extension=False, extension_scope=None,
      serialized_options=None, file=DESCRIPTOR),
    _descriptor.FieldDescriptor(
      name='owners', full_name='types.PBWalletMultiSignData.owners', index=1,
      number=2, type=12, cpp_type=9, label=3,
      has_default_value=False, default_value=[],
      message_type=None, enum_type=None, containing_type=None,
      is_extension=False, extension_scope=None,
      serialized_options=None, file=DESCRIPTOR),
  ],
  extensions=[
  ],
  nested_types=[],
  enum_types=[
  ],
  serialized_options=None,
  is_extendable=False,
  syntax='proto2',
  extension_ranges=[],
  oneofs=[
  ],
  serialized_start=1214,
  serialized_end=1282,
)


_PBTHREEBOTRECORD = _descriptor.Descriptor(
  name='PBThreeBotRecord',
  full_name='types.PBThreeBotRecord',
  filename=None,
  file=DESCRIPTOR,
  containing_type=None,
  fields=[
    _descriptor.FieldDescriptor(
      name='id', full_name='types.PBThreeBotRecord.id', index=0,
      number=1, type=13, cpp_type=3, label=2,
      has_default_value=False, default_value=0,
      message_type=None, enum_type=None, containing_type=None,
      is_extension=False, extension_scope=None,
      serialized_options=None, file=DESCRIPTOR),
    _descriptor.FieldDescriptor(
      name='network_addresses', full_name='types.PBThreeBotRecord.network_addresses', index=1,
      number=2, type=12, cpp_type=9, label=2,
      has_default_value=False, default_value=_b(""),
      message_type=None, enum_type=None, containing_type=None,
      is_extension=False, extension_scope=None,
      serialized_options=None, file=DESCRIPTOR),
    _descriptor.FieldDescriptor(
      name='names', full_name='types.PBThreeBotRecord.names', index=2,
      number=3, type=12, cpp_type=9, label=2,
      has_default_value=False, default_value=_b(""),
      message_type=None, enum_type=None, containing_type=None,
      is_extension=False, extension_scope=None,
      serialized_options=None, file=DESCRIPTOR),
    _descriptor.FieldDescriptor(
      name='expiration_time', full_name='types.PBThreeBotRecord.expiration_time', index=3,
      number=4, type=12, cpp_type=9, label=2,
      has_default_value=False, default_value=_b(""),
      message_type=None, enum_type=None, containing_type=None,
      is_extension=False, extension_scope=None,
      serialized_options=None, file=DESCRIPTOR),
    _descriptor.FieldDescriptor(
      name='public_key', full_name='types.PBThreeBotRecord.public_key', index=4,
      number=5, type=12, cpp_type=9, label=2,
      has_default_value=False, default_value=_b(""),
      message_type=None, enum_type=None, containing_type=None,
      is_extension=False, extension_scope=None,
      serialized_options=None, file=DESCRIPTOR),
  ],
  extensions=[
  ],
  nested_types=[],
  enum_types=[
  ],
  serialized_options=None,
  is_extendable=False,
  syntax='proto2',
  extension_ranges=[],
  oneofs=[
  ],
  serialized_start=1284,
  serialized_end=1401,
)

_PBWALLET.fields_by_name['balance_unlocked'].message_type = _PBWALLETUNLOCKEDBALANCE
_PBWALLET.fields_by_name['balance_locked'].message_type = _PBWALLETLOCKEDBALANCE
_PBWALLET.fields_by_name['multisign_data'].message_type = _PBWALLETMULTISIGNDATA
_PBWALLETUNLOCKEDBALANCE_OUTPUTSENTRY.fields_by_name['value'].message_type = _PBWALLETUNLOCKEDOUTPUT
_PBWALLETUNLOCKEDBALANCE_OUTPUTSENTRY.containing_type = _PBWALLETUNLOCKEDBALANCE
_PBWALLETUNLOCKEDBALANCE.fields_by_name['outputs'].message_type = _PBWALLETUNLOCKEDBALANCE_OUTPUTSENTRY
_PBWALLETLOCKEDBALANCE_OUTPUTSENTRY.fields_by_name['value'].message_type = _PBWALLETLOCKEDOUTPUT
_PBWALLETLOCKEDBALANCE_OUTPUTSENTRY.containing_type = _PBWALLETLOCKEDBALANCE
_PBWALLETLOCKEDBALANCE.fields_by_name['outputs'].message_type = _PBWALLETLOCKEDBALANCE_OUTPUTSENTRY
DESCRIPTOR.message_types_by_name['PBNetworkStats'] = _PBNETWORKSTATS
DESCRIPTOR.message_types_by_name['PBWallet'] = _PBWALLET
DESCRIPTOR.message_types_by_name['PBWalletUnlockedBalance'] = _PBWALLETUNLOCKEDBALANCE
DESCRIPTOR.message_types_by_name['PBWalletUnlockedOutput'] = _PBWALLETUNLOCKEDOUTPUT
DESCRIPTOR.message_types_by_name['PBWalletLockedBalance'] = _PBWALLETLOCKEDBALANCE
DESCRIPTOR.message_types_by_name['PBWalletLockedOutput'] = _PBWALLETLOCKEDOUTPUT
DESCRIPTOR.message_types_by_name['PBWalletMultiSignData'] = _PBWALLETMULTISIGNDATA
DESCRIPTOR.message_types_by_name['PBThreeBotRecord'] = _PBTHREEBOTRECORD
_sym_db.RegisterFileDescriptor(DESCRIPTOR)

PBNetworkStats = _reflection.GeneratedProtocolMessageType('PBNetworkStats', (_message.Message,), dict(
  DESCRIPTOR = _PBNETWORKSTATS,
  __module__ = 'types_pb2'
  # @@protoc_insertion_point(class_scope:types.PBNetworkStats)
  ))
_sym_db.RegisterMessage(PBNetworkStats)

PBWallet = _reflection.GeneratedProtocolMessageType('PBWallet', (_message.Message,), dict(
  DESCRIPTOR = _PBWALLET,
  __module__ = 'types_pb2'
  # @@protoc_insertion_point(class_scope:types.PBWallet)
  ))
_sym_db.RegisterMessage(PBWallet)

PBWalletUnlockedBalance = _reflection.GeneratedProtocolMessageType('PBWalletUnlockedBalance', (_message.Message,), dict(

  OutputsEntry = _reflection.GeneratedProtocolMessageType('OutputsEntry', (_message.Message,), dict(
    DESCRIPTOR = _PBWALLETUNLOCKEDBALANCE_OUTPUTSENTRY,
    __module__ = 'types_pb2'
    # @@protoc_insertion_point(class_scope:types.PBWalletUnlockedBalance.OutputsEntry)
    ))
  ,
  DESCRIPTOR = _PBWALLETUNLOCKEDBALANCE,
  __module__ = 'types_pb2'
  # @@protoc_insertion_point(class_scope:types.PBWalletUnlockedBalance)
  ))
_sym_db.RegisterMessage(PBWalletUnlockedBalance)
_sym_db.RegisterMessage(PBWalletUnlockedBalance.OutputsEntry)

PBWalletUnlockedOutput = _reflection.GeneratedProtocolMessageType('PBWalletUnlockedOutput', (_message.Message,), dict(
  DESCRIPTOR = _PBWALLETUNLOCKEDOUTPUT,
  __module__ = 'types_pb2'
  # @@protoc_insertion_point(class_scope:types.PBWalletUnlockedOutput)
  ))
_sym_db.RegisterMessage(PBWalletUnlockedOutput)

PBWalletLockedBalance = _reflection.GeneratedProtocolMessageType('PBWalletLockedBalance', (_message.Message,), dict(

  OutputsEntry = _reflection.GeneratedProtocolMessageType('OutputsEntry', (_message.Message,), dict(
    DESCRIPTOR = _PBWALLETLOCKEDBALANCE_OUTPUTSENTRY,
    __module__ = 'types_pb2'
    # @@protoc_insertion_point(class_scope:types.PBWalletLockedBalance.OutputsEntry)
    ))
  ,
  DESCRIPTOR = _PBWALLETLOCKEDBALANCE,
  __module__ = 'types_pb2'
  # @@protoc_insertion_point(class_scope:types.PBWalletLockedBalance)
  ))
_sym_db.RegisterMessage(PBWalletLockedBalance)
_sym_db.RegisterMessage(PBWalletLockedBalance.OutputsEntry)

PBWalletLockedOutput = _reflection.GeneratedProtocolMessageType('PBWalletLockedOutput', (_message.Message,), dict(
  DESCRIPTOR = _PBWALLETLOCKEDOUTPUT,
  __module__ = 'types_pb2'
  # @@protoc_insertion_point(class_scope:types.PBWalletLockedOutput)
  ))
_sym_db.RegisterMessage(PBWalletLockedOutput)

PBWalletMultiSignData = _reflection.GeneratedProtocolMessageType('PBWalletMultiSignData', (_message.Message,), dict(
  DESCRIPTOR = _PBWALLETMULTISIGNDATA,
  __module__ = 'types_pb2'
  # @@protoc_insertion_point(class_scope:types.PBWalletMultiSignData)
  ))
_sym_db.RegisterMessage(PBWalletMultiSignData)

PBThreeBotRecord = _reflection.GeneratedProtocolMessageType('PBThreeBotRecord', (_message.Message,), dict(
  DESCRIPTOR = _PBTHREEBOTRECORD,
  __module__ = 'types_pb2'
  # @@protoc_insertion_point(class_scope:types.PBThreeBotRecord)
  ))
_sym_db.RegisterMessage(PBThreeBotRecord)


_PBWALLETUNLOCKEDBALANCE_OUTPUTSENTRY._options = None
_PBWALLETLOCKEDBALANCE_OUTPUTSENTRY._options = None
# @@protoc_insertion_point(module_scope)
