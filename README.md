# go-image-unpacker

Utility to unpack an image from a binary file.

Expected input file format: uint16 width, uint16 height, float32 r, float32 g, float32 b, float32 r, float32 g, float32 b, ...

r, g, b expected to be in range [0, 1].

Throws an error if width or height is greater than 8192 because, yikes, that's a big image and is probably an error.
