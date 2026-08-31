# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [1.0.1] - 2026-08-30

### Fixed / Changed

- Handle interface fields, cyclic/shared references, and oversized keep lengths safely.
- Mask a shared node once per call and document partial-write/reference behavior.
- Align LICENSE and current documentation with MIT, as confirmed by the maintainer; preserve existing copyright notices.

## [Unreleased]

### Fixed

- Include the full nested Go field/index path in configuration errors, preserving
  first-visit semantics for shared objects and partial masking on errors.

## [1.0.0] - 2026-02-XX

### Added
- 初始版本发布
- struct tag 驱动脱敏：`mask:"phone"` / `mask:"3,4"` / `mask:"-"` / `mask:"*"`
- 内置格式：phone / idcard / bankcard / email / name / address / password
- 通用保留格式（前 n 后 m 字符）
- 嵌套递归：结构体、结构体指针、结构体切片；顶层切片批量脱敏
- 按类型缓存脱敏配置，读路径无锁（atomic.Pointer 写时复制）
- 泛型入口 MaskOf
- 自定义格式 RegisterMask（注册后即时生效）
- 按 rune 处理，中文/emoji 不截断
