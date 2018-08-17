package types

// Code generated by github.com/tinylib/msgp DO NOT EDIT.

import (
	"github.com/tinylib/msgp/msgp"
)

// DecodeMsg implements msgp.Decodable
func (z *NetworkStats) DecodeMsg(dc *msgp.Reader) (err error) {
	var field []byte
	_ = field
	var zb0001 uint32
	zb0001, err = dc.ReadMapHeader()
	if err != nil {
		return
	}
	for zb0001 > 0 {
		zb0001--
		field, err = dc.ReadMapKeyPtr()
		if err != nil {
			return
		}
		switch msgp.UnsafeString(field) {
		case "timestamp":
			err = z.Timestamp.DecodeMsg(dc)
			if err != nil {
				return
			}
		case "blockHeight":
			err = z.BlockHeight.DecodeMsg(dc)
			if err != nil {
				return
			}
		case "txCount":
			z.TransactionCount, err = dc.ReadUint64()
			if err != nil {
				return
			}
		case "coinCreationTxCount":
			z.CoinCreationTransactionCount, err = dc.ReadUint64()
			if err != nil {
				return
			}
		case "valueTxCount":
			z.ValueTransactionCount, err = dc.ReadUint64()
			if err != nil {
				return
			}
		case "coinOutputCount":
			z.CointOutputCount, err = dc.ReadUint64()
			if err != nil {
				return
			}
		case "lockedCoinOutputCount":
			z.LockedCointOutputCount, err = dc.ReadUint64()
			if err != nil {
				return
			}
		case "coinInputCount":
			z.CointInputCount, err = dc.ReadUint64()
			if err != nil {
				return
			}
		case "minerPayoutCount":
			z.MinerPayoutCount, err = dc.ReadUint64()
			if err != nil {
				return
			}
		case "txFeeCount":
			z.TransactionFeeCount, err = dc.ReadUint64()
			if err != nil {
				return
			}
		case "minerPayouts":
			err = z.MinerPayouts.DecodeMsg(dc)
			if err != nil {
				return
			}
		case "txFees":
			err = z.TransactionFees.DecodeMsg(dc)
			if err != nil {
				return
			}
		case "coins":
			err = z.Coins.DecodeMsg(dc)
			if err != nil {
				return
			}
		case "lockedCoins":
			err = z.LockedCoins.DecodeMsg(dc)
			if err != nil {
				return
			}
		default:
			err = dc.Skip()
			if err != nil {
				return
			}
		}
	}
	return
}

// EncodeMsg implements msgp.Encodable
func (z *NetworkStats) EncodeMsg(en *msgp.Writer) (err error) {
	// map header, size 14
	// write "timestamp"
	err = en.Append(0x8e, 0xa9, 0x74, 0x69, 0x6d, 0x65, 0x73, 0x74, 0x61, 0x6d, 0x70)
	if err != nil {
		return
	}
	err = z.Timestamp.EncodeMsg(en)
	if err != nil {
		return
	}
	// write "blockHeight"
	err = en.Append(0xab, 0x62, 0x6c, 0x6f, 0x63, 0x6b, 0x48, 0x65, 0x69, 0x67, 0x68, 0x74)
	if err != nil {
		return
	}
	err = z.BlockHeight.EncodeMsg(en)
	if err != nil {
		return
	}
	// write "txCount"
	err = en.Append(0xa7, 0x74, 0x78, 0x43, 0x6f, 0x75, 0x6e, 0x74)
	if err != nil {
		return
	}
	err = en.WriteUint64(z.TransactionCount)
	if err != nil {
		return
	}
	// write "coinCreationTxCount"
	err = en.Append(0xb3, 0x63, 0x6f, 0x69, 0x6e, 0x43, 0x72, 0x65, 0x61, 0x74, 0x69, 0x6f, 0x6e, 0x54, 0x78, 0x43, 0x6f, 0x75, 0x6e, 0x74)
	if err != nil {
		return
	}
	err = en.WriteUint64(z.CoinCreationTransactionCount)
	if err != nil {
		return
	}
	// write "valueTxCount"
	err = en.Append(0xac, 0x76, 0x61, 0x6c, 0x75, 0x65, 0x54, 0x78, 0x43, 0x6f, 0x75, 0x6e, 0x74)
	if err != nil {
		return
	}
	err = en.WriteUint64(z.ValueTransactionCount)
	if err != nil {
		return
	}
	// write "coinOutputCount"
	err = en.Append(0xaf, 0x63, 0x6f, 0x69, 0x6e, 0x4f, 0x75, 0x74, 0x70, 0x75, 0x74, 0x43, 0x6f, 0x75, 0x6e, 0x74)
	if err != nil {
		return
	}
	err = en.WriteUint64(z.CointOutputCount)
	if err != nil {
		return
	}
	// write "lockedCoinOutputCount"
	err = en.Append(0xb5, 0x6c, 0x6f, 0x63, 0x6b, 0x65, 0x64, 0x43, 0x6f, 0x69, 0x6e, 0x4f, 0x75, 0x74, 0x70, 0x75, 0x74, 0x43, 0x6f, 0x75, 0x6e, 0x74)
	if err != nil {
		return
	}
	err = en.WriteUint64(z.LockedCointOutputCount)
	if err != nil {
		return
	}
	// write "coinInputCount"
	err = en.Append(0xae, 0x63, 0x6f, 0x69, 0x6e, 0x49, 0x6e, 0x70, 0x75, 0x74, 0x43, 0x6f, 0x75, 0x6e, 0x74)
	if err != nil {
		return
	}
	err = en.WriteUint64(z.CointInputCount)
	if err != nil {
		return
	}
	// write "minerPayoutCount"
	err = en.Append(0xb0, 0x6d, 0x69, 0x6e, 0x65, 0x72, 0x50, 0x61, 0x79, 0x6f, 0x75, 0x74, 0x43, 0x6f, 0x75, 0x6e, 0x74)
	if err != nil {
		return
	}
	err = en.WriteUint64(z.MinerPayoutCount)
	if err != nil {
		return
	}
	// write "txFeeCount"
	err = en.Append(0xaa, 0x74, 0x78, 0x46, 0x65, 0x65, 0x43, 0x6f, 0x75, 0x6e, 0x74)
	if err != nil {
		return
	}
	err = en.WriteUint64(z.TransactionFeeCount)
	if err != nil {
		return
	}
	// write "minerPayouts"
	err = en.Append(0xac, 0x6d, 0x69, 0x6e, 0x65, 0x72, 0x50, 0x61, 0x79, 0x6f, 0x75, 0x74, 0x73)
	if err != nil {
		return
	}
	err = z.MinerPayouts.EncodeMsg(en)
	if err != nil {
		return
	}
	// write "txFees"
	err = en.Append(0xa6, 0x74, 0x78, 0x46, 0x65, 0x65, 0x73)
	if err != nil {
		return
	}
	err = z.TransactionFees.EncodeMsg(en)
	if err != nil {
		return
	}
	// write "coins"
	err = en.Append(0xa5, 0x63, 0x6f, 0x69, 0x6e, 0x73)
	if err != nil {
		return
	}
	err = z.Coins.EncodeMsg(en)
	if err != nil {
		return
	}
	// write "lockedCoins"
	err = en.Append(0xab, 0x6c, 0x6f, 0x63, 0x6b, 0x65, 0x64, 0x43, 0x6f, 0x69, 0x6e, 0x73)
	if err != nil {
		return
	}
	err = z.LockedCoins.EncodeMsg(en)
	if err != nil {
		return
	}
	return
}

// Msgsize returns an upper bound estimate of the number of bytes occupied by the serialized message
func (z *NetworkStats) Msgsize() (s int) {
	s = 1 + 10 + z.Timestamp.Msgsize() + 12 + z.BlockHeight.Msgsize() + 8 + msgp.Uint64Size + 20 + msgp.Uint64Size + 13 + msgp.Uint64Size + 16 + msgp.Uint64Size + 22 + msgp.Uint64Size + 15 + msgp.Uint64Size + 17 + msgp.Uint64Size + 11 + msgp.Uint64Size + 13 + z.MinerPayouts.Msgsize() + 7 + z.TransactionFees.Msgsize() + 6 + z.Coins.Msgsize() + 12 + z.LockedCoins.Msgsize()
	return
}

// DecodeMsg implements msgp.Decodable
func (z *Wallet) DecodeMsg(dc *msgp.Reader) (err error) {
	var field []byte
	_ = field
	var zb0001 uint32
	zb0001, err = dc.ReadMapHeader()
	if err != nil {
		return
	}
	for zb0001 > 0 {
		zb0001--
		field, err = dc.ReadMapKeyPtr()
		if err != nil {
			return
		}
		switch msgp.UnsafeString(field) {
		case "balance":
			err = z.Balance.DecodeMsg(dc)
			if err != nil {
				return
			}
		case "multisignAddresses":
			var zb0002 uint32
			zb0002, err = dc.ReadArrayHeader()
			if err != nil {
				return
			}
			if cap(z.MultiSignAddresses) >= int(zb0002) {
				z.MultiSignAddresses = (z.MultiSignAddresses)[:zb0002]
			} else {
				z.MultiSignAddresses = make([]UnlockHash, zb0002)
			}
			for za0001 := range z.MultiSignAddresses {
				err = z.MultiSignAddresses[za0001].DecodeMsg(dc)
				if err != nil {
					return
				}
			}
		case "multisign":
			var zb0003 uint32
			zb0003, err = dc.ReadMapHeader()
			if err != nil {
				return
			}
			for zb0003 > 0 {
				zb0003--
				field, err = dc.ReadMapKeyPtr()
				if err != nil {
					return
				}
				switch msgp.UnsafeString(field) {
				case "owners":
					var zb0004 uint32
					zb0004, err = dc.ReadArrayHeader()
					if err != nil {
						return
					}
					if cap(z.MultiSignData.Owners) >= int(zb0004) {
						z.MultiSignData.Owners = (z.MultiSignData.Owners)[:zb0004]
					} else {
						z.MultiSignData.Owners = make([]UnlockHash, zb0004)
					}
					for za0002 := range z.MultiSignData.Owners {
						err = z.MultiSignData.Owners[za0002].DecodeMsg(dc)
						if err != nil {
							return
						}
					}
				case "signaturesRequired":
					z.MultiSignData.SignaturesRequired, err = dc.ReadUint64()
					if err != nil {
						return
					}
				default:
					err = dc.Skip()
					if err != nil {
						return
					}
				}
			}
		default:
			err = dc.Skip()
			if err != nil {
				return
			}
		}
	}
	return
}

// EncodeMsg implements msgp.Encodable
func (z *Wallet) EncodeMsg(en *msgp.Writer) (err error) {
	// map header, size 3
	// write "balance"
	err = en.Append(0x83, 0xa7, 0x62, 0x61, 0x6c, 0x61, 0x6e, 0x63, 0x65)
	if err != nil {
		return
	}
	err = z.Balance.EncodeMsg(en)
	if err != nil {
		return
	}
	// write "multisignAddresses"
	err = en.Append(0xb2, 0x6d, 0x75, 0x6c, 0x74, 0x69, 0x73, 0x69, 0x67, 0x6e, 0x41, 0x64, 0x64, 0x72, 0x65, 0x73, 0x73, 0x65, 0x73)
	if err != nil {
		return
	}
	err = en.WriteArrayHeader(uint32(len(z.MultiSignAddresses)))
	if err != nil {
		return
	}
	for za0001 := range z.MultiSignAddresses {
		err = z.MultiSignAddresses[za0001].EncodeMsg(en)
		if err != nil {
			return
		}
	}
	// write "multisign"
	// map header, size 2
	// write "owners"
	err = en.Append(0xa9, 0x6d, 0x75, 0x6c, 0x74, 0x69, 0x73, 0x69, 0x67, 0x6e, 0x82, 0xa6, 0x6f, 0x77, 0x6e, 0x65, 0x72, 0x73)
	if err != nil {
		return
	}
	err = en.WriteArrayHeader(uint32(len(z.MultiSignData.Owners)))
	if err != nil {
		return
	}
	for za0002 := range z.MultiSignData.Owners {
		err = z.MultiSignData.Owners[za0002].EncodeMsg(en)
		if err != nil {
			return
		}
	}
	// write "signaturesRequired"
	err = en.Append(0xb2, 0x73, 0x69, 0x67, 0x6e, 0x61, 0x74, 0x75, 0x72, 0x65, 0x73, 0x52, 0x65, 0x71, 0x75, 0x69, 0x72, 0x65, 0x64)
	if err != nil {
		return
	}
	err = en.WriteUint64(z.MultiSignData.SignaturesRequired)
	if err != nil {
		return
	}
	return
}

// Msgsize returns an upper bound estimate of the number of bytes occupied by the serialized message
func (z *Wallet) Msgsize() (s int) {
	s = 1 + 8 + z.Balance.Msgsize() + 19 + msgp.ArrayHeaderSize
	for za0001 := range z.MultiSignAddresses {
		s += z.MultiSignAddresses[za0001].Msgsize()
	}
	s += 10 + 1 + 7 + msgp.ArrayHeaderSize
	for za0002 := range z.MultiSignData.Owners {
		s += z.MultiSignData.Owners[za0002].Msgsize()
	}
	s += 19 + msgp.Uint64Size
	return
}

// DecodeMsg implements msgp.Decodable
func (z *WalletBalance) DecodeMsg(dc *msgp.Reader) (err error) {
	var field []byte
	_ = field
	var zb0001 uint32
	zb0001, err = dc.ReadMapHeader()
	if err != nil {
		return
	}
	for zb0001 > 0 {
		zb0001--
		field, err = dc.ReadMapKeyPtr()
		if err != nil {
			return
		}
		switch msgp.UnsafeString(field) {
		case "unlocked":
			err = z.Unlocked.DecodeMsg(dc)
			if err != nil {
				return
			}
		case "locked":
			var zb0002 uint32
			zb0002, err = dc.ReadMapHeader()
			if err != nil {
				return
			}
			for zb0002 > 0 {
				zb0002--
				field, err = dc.ReadMapKeyPtr()
				if err != nil {
					return
				}
				switch msgp.UnsafeString(field) {
				case "total":
					err = z.Locked.Total.DecodeMsg(dc)
					if err != nil {
						return
					}
				case "outputs":
					var zb0003 uint32
					zb0003, err = dc.ReadMapHeader()
					if err != nil {
						return
					}
					if z.Locked.Outputs == nil {
						z.Locked.Outputs = make(WalletLockedOutputMap, zb0003)
					} else if len(z.Locked.Outputs) > 0 {
						for key := range z.Locked.Outputs {
							delete(z.Locked.Outputs, key)
						}
					}
					for zb0003 > 0 {
						zb0003--
						var za0001 string
						var za0002 WalletLockedOutput
						za0001, err = dc.ReadString()
						if err != nil {
							return
						}
						var zb0004 uint32
						zb0004, err = dc.ReadMapHeader()
						if err != nil {
							return
						}
						for zb0004 > 0 {
							zb0004--
							field, err = dc.ReadMapKeyPtr()
							if err != nil {
								return
							}
							switch msgp.UnsafeString(field) {
							case "amount":
								err = za0002.Amount.DecodeMsg(dc)
								if err != nil {
									return
								}
							case "lockedUntil":
								err = za0002.LockedUntil.DecodeMsg(dc)
								if err != nil {
									return
								}
							case "description":
								za0002.Description, err = dc.ReadString()
								if err != nil {
									return
								}
							default:
								err = dc.Skip()
								if err != nil {
									return
								}
							}
						}
						z.Locked.Outputs[za0001] = za0002
					}
				default:
					err = dc.Skip()
					if err != nil {
						return
					}
				}
			}
		default:
			err = dc.Skip()
			if err != nil {
				return
			}
		}
	}
	return
}

// EncodeMsg implements msgp.Encodable
func (z *WalletBalance) EncodeMsg(en *msgp.Writer) (err error) {
	// map header, size 2
	// write "unlocked"
	err = en.Append(0x82, 0xa8, 0x75, 0x6e, 0x6c, 0x6f, 0x63, 0x6b, 0x65, 0x64)
	if err != nil {
		return
	}
	err = z.Unlocked.EncodeMsg(en)
	if err != nil {
		return
	}
	// write "locked"
	// map header, size 2
	// write "total"
	err = en.Append(0xa6, 0x6c, 0x6f, 0x63, 0x6b, 0x65, 0x64, 0x82, 0xa5, 0x74, 0x6f, 0x74, 0x61, 0x6c)
	if err != nil {
		return
	}
	err = z.Locked.Total.EncodeMsg(en)
	if err != nil {
		return
	}
	// write "outputs"
	err = en.Append(0xa7, 0x6f, 0x75, 0x74, 0x70, 0x75, 0x74, 0x73)
	if err != nil {
		return
	}
	err = en.WriteMapHeader(uint32(len(z.Locked.Outputs)))
	if err != nil {
		return
	}
	for za0001, za0002 := range z.Locked.Outputs {
		err = en.WriteString(za0001)
		if err != nil {
			return
		}
		// map header, size 3
		// write "amount"
		err = en.Append(0x83, 0xa6, 0x61, 0x6d, 0x6f, 0x75, 0x6e, 0x74)
		if err != nil {
			return
		}
		err = za0002.Amount.EncodeMsg(en)
		if err != nil {
			return
		}
		// write "lockedUntil"
		err = en.Append(0xab, 0x6c, 0x6f, 0x63, 0x6b, 0x65, 0x64, 0x55, 0x6e, 0x74, 0x69, 0x6c)
		if err != nil {
			return
		}
		err = za0002.LockedUntil.EncodeMsg(en)
		if err != nil {
			return
		}
		// write "description"
		err = en.Append(0xab, 0x64, 0x65, 0x73, 0x63, 0x72, 0x69, 0x70, 0x74, 0x69, 0x6f, 0x6e)
		if err != nil {
			return
		}
		err = en.WriteString(za0002.Description)
		if err != nil {
			return
		}
	}
	return
}

// Msgsize returns an upper bound estimate of the number of bytes occupied by the serialized message
func (z *WalletBalance) Msgsize() (s int) {
	s = 1 + 9 + z.Unlocked.Msgsize() + 7 + 1 + 6 + z.Locked.Total.Msgsize() + 8 + msgp.MapHeaderSize
	if z.Locked.Outputs != nil {
		for za0001, za0002 := range z.Locked.Outputs {
			_ = za0002
			s += msgp.StringPrefixSize + len(za0001) + 1 + 7 + za0002.Amount.Msgsize() + 12 + za0002.LockedUntil.Msgsize() + 12 + msgp.StringPrefixSize + len(za0002.Description)
		}
	}
	return
}

// DecodeMsg implements msgp.Decodable
func (z *WalletLockedBalance) DecodeMsg(dc *msgp.Reader) (err error) {
	var field []byte
	_ = field
	var zb0001 uint32
	zb0001, err = dc.ReadMapHeader()
	if err != nil {
		return
	}
	for zb0001 > 0 {
		zb0001--
		field, err = dc.ReadMapKeyPtr()
		if err != nil {
			return
		}
		switch msgp.UnsafeString(field) {
		case "total":
			err = z.Total.DecodeMsg(dc)
			if err != nil {
				return
			}
		case "outputs":
			var zb0002 uint32
			zb0002, err = dc.ReadMapHeader()
			if err != nil {
				return
			}
			if z.Outputs == nil {
				z.Outputs = make(WalletLockedOutputMap, zb0002)
			} else if len(z.Outputs) > 0 {
				for key := range z.Outputs {
					delete(z.Outputs, key)
				}
			}
			for zb0002 > 0 {
				zb0002--
				var za0001 string
				var za0002 WalletLockedOutput
				za0001, err = dc.ReadString()
				if err != nil {
					return
				}
				var zb0003 uint32
				zb0003, err = dc.ReadMapHeader()
				if err != nil {
					return
				}
				for zb0003 > 0 {
					zb0003--
					field, err = dc.ReadMapKeyPtr()
					if err != nil {
						return
					}
					switch msgp.UnsafeString(field) {
					case "amount":
						err = za0002.Amount.DecodeMsg(dc)
						if err != nil {
							return
						}
					case "lockedUntil":
						err = za0002.LockedUntil.DecodeMsg(dc)
						if err != nil {
							return
						}
					case "description":
						za0002.Description, err = dc.ReadString()
						if err != nil {
							return
						}
					default:
						err = dc.Skip()
						if err != nil {
							return
						}
					}
				}
				z.Outputs[za0001] = za0002
			}
		default:
			err = dc.Skip()
			if err != nil {
				return
			}
		}
	}
	return
}

// EncodeMsg implements msgp.Encodable
func (z *WalletLockedBalance) EncodeMsg(en *msgp.Writer) (err error) {
	// map header, size 2
	// write "total"
	err = en.Append(0x82, 0xa5, 0x74, 0x6f, 0x74, 0x61, 0x6c)
	if err != nil {
		return
	}
	err = z.Total.EncodeMsg(en)
	if err != nil {
		return
	}
	// write "outputs"
	err = en.Append(0xa7, 0x6f, 0x75, 0x74, 0x70, 0x75, 0x74, 0x73)
	if err != nil {
		return
	}
	err = en.WriteMapHeader(uint32(len(z.Outputs)))
	if err != nil {
		return
	}
	for za0001, za0002 := range z.Outputs {
		err = en.WriteString(za0001)
		if err != nil {
			return
		}
		// map header, size 3
		// write "amount"
		err = en.Append(0x83, 0xa6, 0x61, 0x6d, 0x6f, 0x75, 0x6e, 0x74)
		if err != nil {
			return
		}
		err = za0002.Amount.EncodeMsg(en)
		if err != nil {
			return
		}
		// write "lockedUntil"
		err = en.Append(0xab, 0x6c, 0x6f, 0x63, 0x6b, 0x65, 0x64, 0x55, 0x6e, 0x74, 0x69, 0x6c)
		if err != nil {
			return
		}
		err = za0002.LockedUntil.EncodeMsg(en)
		if err != nil {
			return
		}
		// write "description"
		err = en.Append(0xab, 0x64, 0x65, 0x73, 0x63, 0x72, 0x69, 0x70, 0x74, 0x69, 0x6f, 0x6e)
		if err != nil {
			return
		}
		err = en.WriteString(za0002.Description)
		if err != nil {
			return
		}
	}
	return
}

// Msgsize returns an upper bound estimate of the number of bytes occupied by the serialized message
func (z *WalletLockedBalance) Msgsize() (s int) {
	s = 1 + 6 + z.Total.Msgsize() + 8 + msgp.MapHeaderSize
	if z.Outputs != nil {
		for za0001, za0002 := range z.Outputs {
			_ = za0002
			s += msgp.StringPrefixSize + len(za0001) + 1 + 7 + za0002.Amount.Msgsize() + 12 + za0002.LockedUntil.Msgsize() + 12 + msgp.StringPrefixSize + len(za0002.Description)
		}
	}
	return
}

// DecodeMsg implements msgp.Decodable
func (z *WalletLockedOutput) DecodeMsg(dc *msgp.Reader) (err error) {
	var field []byte
	_ = field
	var zb0001 uint32
	zb0001, err = dc.ReadMapHeader()
	if err != nil {
		return
	}
	for zb0001 > 0 {
		zb0001--
		field, err = dc.ReadMapKeyPtr()
		if err != nil {
			return
		}
		switch msgp.UnsafeString(field) {
		case "amount":
			err = z.Amount.DecodeMsg(dc)
			if err != nil {
				return
			}
		case "lockedUntil":
			err = z.LockedUntil.DecodeMsg(dc)
			if err != nil {
				return
			}
		case "description":
			z.Description, err = dc.ReadString()
			if err != nil {
				return
			}
		default:
			err = dc.Skip()
			if err != nil {
				return
			}
		}
	}
	return
}

// EncodeMsg implements msgp.Encodable
func (z *WalletLockedOutput) EncodeMsg(en *msgp.Writer) (err error) {
	// map header, size 3
	// write "amount"
	err = en.Append(0x83, 0xa6, 0x61, 0x6d, 0x6f, 0x75, 0x6e, 0x74)
	if err != nil {
		return
	}
	err = z.Amount.EncodeMsg(en)
	if err != nil {
		return
	}
	// write "lockedUntil"
	err = en.Append(0xab, 0x6c, 0x6f, 0x63, 0x6b, 0x65, 0x64, 0x55, 0x6e, 0x74, 0x69, 0x6c)
	if err != nil {
		return
	}
	err = z.LockedUntil.EncodeMsg(en)
	if err != nil {
		return
	}
	// write "description"
	err = en.Append(0xab, 0x64, 0x65, 0x73, 0x63, 0x72, 0x69, 0x70, 0x74, 0x69, 0x6f, 0x6e)
	if err != nil {
		return
	}
	err = en.WriteString(z.Description)
	if err != nil {
		return
	}
	return
}

// Msgsize returns an upper bound estimate of the number of bytes occupied by the serialized message
func (z *WalletLockedOutput) Msgsize() (s int) {
	s = 1 + 7 + z.Amount.Msgsize() + 12 + z.LockedUntil.Msgsize() + 12 + msgp.StringPrefixSize + len(z.Description)
	return
}

// DecodeMsg implements msgp.Decodable
func (z *WalletLockedOutputMap) DecodeMsg(dc *msgp.Reader) (err error) {
	var zb0003 uint32
	zb0003, err = dc.ReadMapHeader()
	if err != nil {
		return
	}
	if (*z) == nil {
		(*z) = make(WalletLockedOutputMap, zb0003)
	} else if len((*z)) > 0 {
		for key := range *z {
			delete((*z), key)
		}
	}
	for zb0003 > 0 {
		zb0003--
		var zb0001 string
		var zb0002 WalletLockedOutput
		zb0001, err = dc.ReadString()
		if err != nil {
			return
		}
		var field []byte
		_ = field
		var zb0004 uint32
		zb0004, err = dc.ReadMapHeader()
		if err != nil {
			return
		}
		for zb0004 > 0 {
			zb0004--
			field, err = dc.ReadMapKeyPtr()
			if err != nil {
				return
			}
			switch msgp.UnsafeString(field) {
			case "amount":
				err = zb0002.Amount.DecodeMsg(dc)
				if err != nil {
					return
				}
			case "lockedUntil":
				err = zb0002.LockedUntil.DecodeMsg(dc)
				if err != nil {
					return
				}
			case "description":
				zb0002.Description, err = dc.ReadString()
				if err != nil {
					return
				}
			default:
				err = dc.Skip()
				if err != nil {
					return
				}
			}
		}
		(*z)[zb0001] = zb0002
	}
	return
}

// EncodeMsg implements msgp.Encodable
func (z WalletLockedOutputMap) EncodeMsg(en *msgp.Writer) (err error) {
	err = en.WriteMapHeader(uint32(len(z)))
	if err != nil {
		return
	}
	for zb0005, zb0006 := range z {
		err = en.WriteString(zb0005)
		if err != nil {
			return
		}
		// map header, size 3
		// write "amount"
		err = en.Append(0x83, 0xa6, 0x61, 0x6d, 0x6f, 0x75, 0x6e, 0x74)
		if err != nil {
			return
		}
		err = zb0006.Amount.EncodeMsg(en)
		if err != nil {
			return
		}
		// write "lockedUntil"
		err = en.Append(0xab, 0x6c, 0x6f, 0x63, 0x6b, 0x65, 0x64, 0x55, 0x6e, 0x74, 0x69, 0x6c)
		if err != nil {
			return
		}
		err = zb0006.LockedUntil.EncodeMsg(en)
		if err != nil {
			return
		}
		// write "description"
		err = en.Append(0xab, 0x64, 0x65, 0x73, 0x63, 0x72, 0x69, 0x70, 0x74, 0x69, 0x6f, 0x6e)
		if err != nil {
			return
		}
		err = en.WriteString(zb0006.Description)
		if err != nil {
			return
		}
	}
	return
}

// Msgsize returns an upper bound estimate of the number of bytes occupied by the serialized message
func (z WalletLockedOutputMap) Msgsize() (s int) {
	s = msgp.MapHeaderSize
	if z != nil {
		for zb0005, zb0006 := range z {
			_ = zb0006
			s += msgp.StringPrefixSize + len(zb0005) + 1 + 7 + zb0006.Amount.Msgsize() + 12 + zb0006.LockedUntil.Msgsize() + 12 + msgp.StringPrefixSize + len(zb0006.Description)
		}
	}
	return
}

// DecodeMsg implements msgp.Decodable
func (z *WalletMultiSignData) DecodeMsg(dc *msgp.Reader) (err error) {
	var field []byte
	_ = field
	var zb0001 uint32
	zb0001, err = dc.ReadMapHeader()
	if err != nil {
		return
	}
	for zb0001 > 0 {
		zb0001--
		field, err = dc.ReadMapKeyPtr()
		if err != nil {
			return
		}
		switch msgp.UnsafeString(field) {
		case "owners":
			var zb0002 uint32
			zb0002, err = dc.ReadArrayHeader()
			if err != nil {
				return
			}
			if cap(z.Owners) >= int(zb0002) {
				z.Owners = (z.Owners)[:zb0002]
			} else {
				z.Owners = make([]UnlockHash, zb0002)
			}
			for za0001 := range z.Owners {
				err = z.Owners[za0001].DecodeMsg(dc)
				if err != nil {
					return
				}
			}
		case "signaturesRequired":
			z.SignaturesRequired, err = dc.ReadUint64()
			if err != nil {
				return
			}
		default:
			err = dc.Skip()
			if err != nil {
				return
			}
		}
	}
	return
}

// EncodeMsg implements msgp.Encodable
func (z *WalletMultiSignData) EncodeMsg(en *msgp.Writer) (err error) {
	// map header, size 2
	// write "owners"
	err = en.Append(0x82, 0xa6, 0x6f, 0x77, 0x6e, 0x65, 0x72, 0x73)
	if err != nil {
		return
	}
	err = en.WriteArrayHeader(uint32(len(z.Owners)))
	if err != nil {
		return
	}
	for za0001 := range z.Owners {
		err = z.Owners[za0001].EncodeMsg(en)
		if err != nil {
			return
		}
	}
	// write "signaturesRequired"
	err = en.Append(0xb2, 0x73, 0x69, 0x67, 0x6e, 0x61, 0x74, 0x75, 0x72, 0x65, 0x73, 0x52, 0x65, 0x71, 0x75, 0x69, 0x72, 0x65, 0x64)
	if err != nil {
		return
	}
	err = en.WriteUint64(z.SignaturesRequired)
	if err != nil {
		return
	}
	return
}

// Msgsize returns an upper bound estimate of the number of bytes occupied by the serialized message
func (z *WalletMultiSignData) Msgsize() (s int) {
	s = 1 + 7 + msgp.ArrayHeaderSize
	for za0001 := range z.Owners {
		s += z.Owners[za0001].Msgsize()
	}
	s += 19 + msgp.Uint64Size
	return
}
