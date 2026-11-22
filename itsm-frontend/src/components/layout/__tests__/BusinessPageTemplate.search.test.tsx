/**
 * 共享模板搜索防抖（计划 F-P0-1）。
 *
 * 修复前 debouncedSearch 算了没用、onChange 每键直调 onSearch，消费方
 * （changes/problems/incidents 等）的 fetch effect 依赖该值时每键一发请求。
 * 修复后：防抖 300ms 收敛一次；回车/按钮即时发出，且防抖落地不重复发同一词。
 */
import React from 'react';
import { render, screen, fireEvent, act } from '@/lib/test-utils';
import { BusinessPageTemplate } from '../BusinessPageTemplate';

describe('BusinessPageTemplate 搜索防抖', () => {
  beforeEach(() => {
    jest.useFakeTimers();
  });

  afterEach(() => {
    jest.useRealTimers();
  });

  const type = (input: HTMLElement, value: string) => {
    fireEvent.change(input, { target: { value } });
  };

  // rc-input 用 keyLockRef 防按住回车重复触发：keydown 置锁、keyup 复位。
  // 真实键盘必有 keyup，测试须成对发，否则第一次回车后锁死。
  const pressEnter = (input: HTMLElement) => {
    fireEvent.keyDown(input, { key: 'Enter', keyCode: 13 });
    fireEvent.keyUp(input, { key: 'Enter', keyCode: 13 });
  };

  it('挂载时不触发 onSearch（初值视为已发出）', () => {
    const onSearch = jest.fn();
    render(
      <BusinessPageTemplate
        title="变更管理"
        searchValue="已有词"
        searchPlaceholder="搜索工单"
        onSearch={onSearch}
      />
    );

    act(() => {
      jest.advanceTimersByTime(1000);
    });
    expect(onSearch).not.toHaveBeenCalled();
  });

  it('连续输入收敛为防抖后一次，且带最终词', () => {
    const onSearch = jest.fn();
    render(
      <BusinessPageTemplate title="变更管理" searchPlaceholder="搜索工单" onSearch={onSearch} />
    );
    const input = screen.getByPlaceholderText('搜索工单');

    type(input, 'in');
    act(() => {
      jest.advanceTimersByTime(150);
    });
    type(input, 'inc');
    // 防抖未落地不发
    expect(onSearch).not.toHaveBeenCalled();

    act(() => {
      jest.advanceTimersByTime(300);
    });
    expect(onSearch).toHaveBeenCalledTimes(1);
    expect(onSearch).toHaveBeenCalledWith('inc');
  });

  it('回车即时发出当前词，防抖落地不重复发同一词', () => {
    const onSearch = jest.fn();
    render(
      <BusinessPageTemplate title="变更管理" searchPlaceholder="搜索工单" onSearch={onSearch} />
    );
    const input = screen.getByPlaceholderText('搜索工单');

    type(input, 'change');
    pressEnter(input);
    expect(onSearch).toHaveBeenCalledTimes(1);
    expect(onSearch).toHaveBeenCalledWith('change');

    act(() => {
      jest.advanceTimersByTime(500);
    });
    // 防抖落地值与已发词相同 → 跳过
    expect(onSearch).toHaveBeenCalledTimes(1);
  });

  it('回车发过的词继续修改后，防抖照常发出新词', () => {
    const onSearch = jest.fn();
    render(
      <BusinessPageTemplate title="变更管理" searchPlaceholder="搜索工单" onSearch={onSearch} />
    );
    const input = screen.getByPlaceholderText('搜索工单');

    type(input, 'change');
    pressEnter(input);
    expect(onSearch).toHaveBeenCalledTimes(1);

    type(input, 'change-2');
    act(() => {
      jest.advanceTimersByTime(300);
    });
    expect(onSearch).toHaveBeenCalledTimes(2);
    expect(onSearch).toHaveBeenLastCalledWith('change-2');
  });

  // 父组件回调换身份不能让旧防抖值再次提交。
  it('回车后更换回调身份不回放旧词，防抖到期新词总共只发送一次', () => {
    const first = jest.fn();
    const latest = jest.fn();
    const { rerender } = render(
      <BusinessPageTemplate title="事件管理" searchValue="旧词" searchPlaceholder="搜索工单" onSearch={first} />
    );
    const input = screen.getByPlaceholderText('搜索工单');
    type(input, '新词');
    act(() => { jest.advanceTimersByTime(100); });
    pressEnter(input);
    rerender(
      <BusinessPageTemplate title="事件管理" searchValue="旧词" searchPlaceholder="搜索工单" onSearch={latest} />
    );
    expect(first.mock.calls).toEqual([['新词']]);
    expect(latest).not.toHaveBeenCalled();
    act(() => { jest.advanceTimersByTime(300); });
    expect(first.mock.calls).toEqual([['新词']]);
    expect(latest).not.toHaveBeenCalled();
    type(input, '后续词');
    act(() => { jest.advanceTimersByTime(300); });
    expect(latest.mock.calls).toEqual([['后续词']]);
  });

  it('未传 onSearch 时输入与回车不报错', () => {
    render(<BusinessPageTemplate title="变更管理" searchPlaceholder="搜索工单" />);
    const input = screen.getByPlaceholderText('搜索工单');

    type(input, 'x');
    pressEnter(input);
    act(() => {
      jest.advanceTimersByTime(300);
    });
  });
});
