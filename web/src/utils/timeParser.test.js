// 单元测试:node --test(node 18+ 内置测试运行器,无需额外依赖)
// 运行:npm test  或  node --test src/utils/timeParser.test.js
import { test, describe } from 'node:test'
import assert from 'node:assert/strict'
import { fmtTime, fmtTtl } from './timeParser.js'

describe('fmtTime', () => {
  test('ISO 时间去 T 并截断到秒', () => {
    assert.equal(fmtTime('2026-09-20T10:30:00Z'), '2026-09-20 10:30:00')
    assert.equal(fmtTime('2026-09-20 10:30:00'), '2026-09-20 10:30:00')
  })

  test('空值返回占位符', () => {
    assert.equal(fmtTime(''), '-')
    assert.equal(fmtTime(null), '-')
    assert.equal(fmtTime(undefined), '-')
  })
})

describe('fmtTtl', () => {
  test('非正数与非法输入返回占位符', () => {
    assert.equal(fmtTtl(0), '-')
    assert.equal(fmtTtl(-5), '-')
    assert.equal(fmtTtl(null), '-')
    assert.equal(fmtTtl(undefined), '-')
    assert.equal(fmtTtl('abc'), '-')
  })

  test('单一单位', () => {
    assert.equal(fmtTtl(30), '30秒')
    assert.equal(fmtTtl(60), '1分钟')
    assert.equal(fmtTtl(3600), '1小时')
    assert.equal(fmtTtl(86400), '1天')
  })

  test('最多保留两个非零单位', () => {
    assert.equal(fmtTtl(90), '1分钟30秒')
    assert.equal(fmtTtl(3661), '1小时1分钟')
    assert.equal(fmtTtl(90000), '1天1小时')
  })

  test('丢弃第三个及以后的单位', () => {
    // 1天1小时1分1秒 -> 只保留前两个非零单位
    assert.equal(fmtTtl(86400 + 3600 + 60 + 1), '1天1小时')
  })

  test('字符串数字同样可解析', () => {
    assert.equal(fmtTtl('120'), '2分钟')
  })

  test('不足 1 秒回落为秒', () => {
    assert.equal(fmtTtl(0.5), '0.5秒')
  })
})
