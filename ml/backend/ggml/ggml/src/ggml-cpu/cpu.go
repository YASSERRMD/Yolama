package cpu

// #cgo CFLAGS: -O3 -Wno-implicit-function-declaration
// #cgo CXXFLAGS: -std=c++17
// #cgo CPPFLAGS: -I${SRCDIR}/amx -I${SRCDIR}/llamafile -I${SRCDIR}/.. -I${SRCDIR}/../../include
// #cgo CPPFLAGS: -DNDEBUG -DGGML_USE_LLAMAFILE
// #cgo linux CPPFLAGS: -D_GNU_SOURCE
// #cgo darwin,arm64 CPPFLAGS: -DGGML_USE_ACCELERATE -DACCELERATE_NEW_LAPACK -DACCELERATE_LAPACK_ILP64
// #cgo darwin,arm64 LDFLAGS: -framework Accelerate
import "C"
import _ "github.com/YASSERRMD/Yolama/ml/backend/ggml/ggml/src/ggml-cpu/llamafile"
