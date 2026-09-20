// 单元测试:node --test(node 18+ 内置测试运行器,无需额外依赖)
// 运行:npm test  或  node --test src/utils/cron.test.js
import { test, describe } from 'node:test'
import assert from 'node:assert/strict'
import { cronToText, WEEK_CN } from './cron.js'

describe('cronToText', () => {
  test('空值返回占位符', () => {
    assert.equal(cronToText(''), '-')
    assert.equal(cronToText(null), '-')
    assert.equal(cronToText(undefined), '-')
  })

  test('非 5 段原样返回', () => {
    assert.equal(cronToText('0 9 *'), '0 9 *')
    assert.equal(cronToText('* * * * * *'), '* * * * * *')
  })

  test('含不支持字段原样返回', () => {
    assert.equal(cronToText('99 * * * *'), '99 * * * *')
    assert.equal(cronToText('0 9 * * MON'), '0 9 * * MON')
  })

  test('纯分钟级', () => {
    assert.equal(cronToText('* * * * *'), '每分钟')
    assert.equal(cronToText('*/2 * * * *'), '每2分钟')
    assert.equal(cronToText('*/30 * * * *'), '每30分钟')
    assert.equal(cronToText('15 * * * *'), '每小时第 15 分钟')
  })

  test('每天固定时间', () => {
    assert.equal(cronToText('0 9 * * *'), '每天 09:00')
    assert.equal(cronToText('30 8 * * *'), '每天 08:30')
    assert.equal(cronToText('0 12 * * *'), '每天 12:00')
  })

  test('整点每分钟', () => {
    assert.equal(cronToText('* 9 * * *'), '每天 09点每分钟')
  })

  test('工作日定时', () => {
    assert.equal(cronToText('30 8 * * 1-5'), '周一~周五 08:30')
  })

  test('周日(0 与 7 等价)', () => {
    assert.equal(cronToText('0 12 * * 0'), '周日 12:00')
    assert.equal(cronToText('0 12 * * 7'), '周日 12:00')
  })

  test('每月固定日期', () => {
    assert.equal(cronToText('0 0 1 * *'), '1日 00:00')
    assert.equal(cronToText('0 0 1,15 * *'), '1日、15日 00:00')
  })

  test('指定月份', () => {
    assert.equal(cronToText('0 9 1 6 *'), '6月 1日 09:00')
  })

  test('WEEK_CN 映射完整', () => {
    assert.equal(WEEK_CN[0], '周日')
    assert.equal(WEEK_CN[7], '周日')
    assert.equal(WEEK_CN[3], '周三')
  })
})
