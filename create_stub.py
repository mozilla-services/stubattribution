import sys
import struct
import mmap

def create_stub(stubtype):
    """Returns a bytearray of a fake stub installer.

    This is done by reversing the stub modification code.
    It's not ideal, but it makes for something to do minimal testing with.
    """
    #### Calculate values ####

    # put header after offset location of 0x3C:0x40
    pe_header_offset = 0x40
    # from code
    optional_header_offset = pe_header_offset + 24
    # reverse:
    # if pe_magic_number == 0x10b:
    #     # 32-bit
    #     cert_dir_entry_offset = optional_header_offset + 128
    # elif pe_magic_number == 0x20b:
    #     # 64-bit. Certain header fields are wider.
    #     cert_dir_entry_offset = optional_header_offset + 144
    if stubtype == '32':
        pe_magic_number = 0x10b
        cert_dir_entry_offset = optional_header_offset + 128
    elif stubtype == '64':
        pe_magic_number = 0x20b
        cert_dir_entry_offset = optional_header_offset + 144
    else:
        raise Exception('invalid second arg, must be 32 or 64')
    # from code
    tag = b"__MOZCUSTOM__:"
    # spec: there exists a 1024-byte space which begins with the tag "__MOZCUSTOM__:"
    data = chr(0)*(1024-len(tag))
    # put table after cert_dir_entry (which is 8 bytes long)
    cert_table_offset = cert_dir_entry_offset + 8
    # table must contain tag+value
    cert_table_size = len(tag) + len(data)
    # initialize minimum size stub file with 0's
    stub = bytearray(chr(0)*(cert_table_offset+cert_table_size))

    #### Populate the fake stub ####

    # reverse: pe_header_offset = struct.unpack("<I", stub[0x3C:0x40])[0]
    stub[0x3C:0x40] = struct.pack("<I", pe_header_offset)
    # reverse: pe_magic_number = struct.unpack("<H", stub[optional_header_offset:optional_header_offset+2])[0]
    stub[optional_header_offset:optional_header_offset+2] = struct.pack("<H",pe_magic_number)
    # reverse: cert_table_offset = struct.unpack("<I", stub[cert_dir_entry_offset:cert_dir_entry_offset+4])[0]
    stub[cert_dir_entry_offset:cert_dir_entry_offset+4] = struct.pack("<I", cert_table_offset)
    # reverse: cert_table_size = struct.unpack("<I", stub[cert_dir_entry_offset+4:cert_dir_entry_offset+8])[0]
    stub[cert_dir_entry_offset+4:cert_dir_entry_offset+8] = struct.pack("<I", cert_table_size)
    # reverse: tag_index = stub.find(tag, cert_table_offset, cert_table_offset + cert_table_size)
    stub[cert_table_offset:cert_table_offset+len(tag)] = tag
    # reverse: stub[tag_index+len(tag):tag_index+len(tag)+len(data)] = data
    stub[cert_table_offset+len(tag):cert_table_offset+len(tag)+len(data)] = data

    return stub

if __name__ == '__main__':
    # Receive arguments
    filepath = sys.argv[1]
    stubtype = sys.argv[2]
    # Create stub
    stub = create_stub(stubtype)
    # Write stub ####
    with open(filepath, 'w+b') as f:
        f.write(bytes(stub))
