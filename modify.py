import mmap
import struct
import sys

class AttributionException(Exception):
    pass

def write_attribution_data(mapped, data):
    """Insert data into a prepared certificate in a signed PE file.

    Parameters are a stub installer in a bytearray and an attribution code as a string.

    Returns False if the file isn't a valid PE file, or if the necessary
    certificate was not found.

    This function assumes that somewhere in the given file's certificate table
    there exists a 1024-byte space which begins with the tag "__MOZCUSTOM__:".
    The given data will be inserted into the file following this tag.

    We don't bother updating the optional header checksum.
    Windows doesn't check it for executables, only drivers and certain DLL's.
    """
    # Get the location of the PE header and the optional header
    pe_header_offset = struct.unpack("<I", mapped[0x3C:0x40])[0]
    optional_header_offset = pe_header_offset + 24

    # Look up the magic number in the optional header,
    # so we know if we have a 32 or 64-bit executable.
    # We need to know that so that we can find the data directories.
    pe_magic_number = struct.unpack(
        "<H", mapped[optional_header_offset:optional_header_offset+2])[0]
    if pe_magic_number == 0x10b:
        # 32-bit
        cert_dir_entry_offset = optional_header_offset + 128
    elif pe_magic_number == 0x20b:
        # 64-bit. Certain header fields are wider.
        cert_dir_entry_offset = optional_header_offset + 144
    else:
        raise AttributionException('mapped is not in a known PE format')

    # The certificate table offset and length give us the valid range
    # to search through for where we should put our data.
    cert_table_offset = struct.unpack(
        "<I", mapped[cert_dir_entry_offset:cert_dir_entry_offset+4])[0]
    cert_table_size = struct.unpack(
        "<I", mapped[cert_dir_entry_offset+4:cert_dir_entry_offset+8])[0]

    if cert_table_offset == 0 or cert_table_size == 0:
        # The file isn't signed
        raise AttributionException('mapped is not signed')

    tag = b"__MOZCUSTOM__:"
    tag_index = mapped.find(tag, cert_table_offset,
        cert_table_offset + cert_table_size)
    if tag_index == -1:
        raise AttributionException('mapped does not contain dummy cert')

    if sys.version_info >= (3,):
        data = data.encode("utf-8")

    mapped[tag_index+len(tag):tag_index+len(tag)+len(data)] = data


if __name__ == "__main__":
    # read mapped installer as bytearray so it can be modified
    with open(sys.argv[1], 'r+b') as f:
        mapped = bytearray(f.read())
    write_attribution_data(mapped, sys.argv[2])
    with open(sys.argv[1], 'w+b') as f:
        f.write(mapped)
