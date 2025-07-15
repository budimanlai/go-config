# Changelog

Semua perubahan penting dalam proyek ini akan didokumentasikan dalam file ini.

Format berdasarkan [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
dan proyek ini mengikuti [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [0.0.4] - 2025-07-15

### 🚀 Fitur Utama
- **Konversi Type-Aware**: Konversi tipe cerdas berdasarkan tipe field struct
- **Performance Caching**: Peningkatan kecepatan 6.45x dengan reflection result caching
- **Production Monitoring**: Built-in statistics dan monitoring APIs (`GetStats()`)
- **Resource Management**: Cleanup yang proper dengan method `Close()`
- **Memory Optimization**: Pengurangan 84% pada alokasi, 80% pengurangan penggunaan memory

### ✨ Ditambahkan
- Method `Close()` untuk resource cleanup yang proper
- Method `GetStats()` untuk production monitoring
- Method `ClearCache()` untuk cache management
- Type-aware struct mapping di `MapToStructNested()`
- Reflection result caching untuk performa
- Pre-allocated memory pools untuk optimasi
- Enhanced concurrent access safety dengan RWMutex
- Production-ready error handling dan recovery

### 🐛 Diperbaiki
- **Kritis**: String field yang berisi angka salah di-parse sebagai integer
- **Kritis**: Array element type mapping hanya bekerja untuk index `.0`
- Memory leak di file watcher dan goroutine
- Race condition dalam concurrent access scenario
- Operasi string yang tidak efisien menyebabkan performance bottleneck

### 🔧 Diubah
- Enhanced `MapToStructNested()` dengan type-aware conversion
- Improved array/slice processing untuk ukuran index apapun
- Optimasi memory allocation di hot code path
- Error message yang lebih baik dengan informasi kontekstual
- Thread-safe operation untuk semua public method

### ⚠️ Breaking Changes
- **Resource Management**: Method `Close()` harus dipanggil untuk cleanup yang proper
- Enhanced error handling mungkin mengubah format error message

### 📈 Performa
- **MapToStruct (cold)**: ~57µs → 8.9µs (6.4x lebih cepat)
- **MapToStruct (cached)**: ~57µs → 1.3µs (43.8x lebih cepat)
- **Memory allocation**: 183/op → 29/op (pengurangan 84%)
- **Memory usage**: 10,113B → 2,065B (pengurangan 80%)
- **Concurrent access**: 0.620µs/op dengan skalabilitas excellent

### 🧪 Testing
- Ditambahkan 6 production feature test baru
- Comprehensive benchmark suite dengan memory profiling
- Concurrent access testing dengan multiple goroutine
- Large dataset performance validation (1000+ item)
- Memory leak dan resource cleanup verification

## [0.0.3] - Release Sebelumnya
- Basic configuration reading
- Hot reload functionality
- Array dan object support
- Thread-safe operation

## [0.0.2] - Release Sebelumnya
- JSON format support
- Nested configuration key
- Basic struct mapping

## [0.0.1] - Initial Release
- INI format support
- Basic key-value reading
- File watching capabilities
