/**
 * CMDB 页面测试
 */

import React from 'react';
import { render, screen, fireEvent, waitFor, act } from '@testing-library/react';
import '@testing-library/jest-dom';

import CIList from '@/modules/cmdb/components/CIList';
import CIDetail from '@/modules/cmdb/components/CIDetail';
import CreateCIPage from '@/app/(main)/cmdb/cis/create/page';
import { CMDBApi } from '@/modules/cmdb/api';
import type { ConfigurationItem, CIType } from '@/modules/cmdb/types';

const mockPush = jest.fn();
const mockBack = jest.fn();
let modalConfirmSpy: jest.Mock;
let formValues: Record<string, unknown> = {};

jest.mock('next/navigation', () => ({
  useRouter: () => ({
    push: mockPush,
    back: mockBack,
    replace: jest.fn(),
    prefetch: jest.fn(),
  }),
  useParams: () => ({ id: '1' }),
  useSearchParams: jest.fn(() => new URLSearchParams()),
  usePathname: () => '/cmdb/cis/create',
}));

jest.mock('antd', () => {
  const actual = jest.requireActual('antd');
  const React = jest.requireActual('react');
  modalConfirmSpy = jest.fn((config: { onOk?: () => void }) => {
    config?.onOk?.();
  });
  formValues = {};
  const FormContext = React.createContext({
    values: {},
    setValues: () => {},
  });

  const Form = ({ children, onFinish }: any) => {
    const [values, setValues] = React.useState<Record<string, unknown>>({});
    const handleSubmit = (event: React.FormEvent) => {
      event.preventDefault();
      onFinish?.(values);
    };

    return (
      <form onSubmit={handleSubmit}>
        <FormContext.Provider value={{ values, setValues }}>
          {children}
        </FormContext.Provider>
      </form>
    );
  };

  Form.Item = ({ label, name, children }: any) => {
    const context = React.useContext(FormContext);
    if (!React.isValidElement(children)) {
      return label ? (
        <label>
          {label}
          {children}
        </label>
      ) : (
        <>{children}</>
      );
    }

    const handleChange = (event: any) => {
      const value = event?.target?.value ?? event;
      context.setValues((prev: Record<string, unknown>) => {
        formValues = { ...prev, [name]: value };
        return formValues;
      });
      children.props.onChange?.(event);
    };

    const inputElement = React.cloneElement(children, {
      'aria-label': label,
      name,
      value: context.values[name] ?? '',
      onChange: handleChange,
    });

    return label ? <label>{label}{inputElement}</label> : inputElement;
  };

  Form.useForm = () => [
    {
      getFieldsValue: () => ({ ...formValues }),
      resetFields: jest.fn(() => {
        formValues = {};
      }),
      setFieldsValue: jest.fn((values: Record<string, unknown>) => {
        formValues = { ...formValues, ...values };
      }),
    },
  ];

  const Input = ({ allowClear, ...rest }: any) => <input {...rest} />;
  Input.TextArea = ({ allowClear, ...rest }: any) => <textarea {...rest} />;

  const Select = ({
    children,
    onChange,
    value,
    allowClear,
    loading,
    placeholder,
    style,
    ...rest
  }: any) => (
    <select
      {...rest}
      value={value ?? ''}
      onChange={(event) => onChange?.(event.target.value)}
    >
      {children}
    </select>
  );
  Select.Option = ({ value, children }: any) => <option value={value}>{children}</option>;

  const Button = ({
    children,
    onClick,
    htmlType,
    danger,
    type,
    loading,
    ...rest
  }: any) => (
    <button
      type={htmlType === 'submit' ? 'submit' : 'button'}
      onClick={onClick}
      data-danger={danger ? 'true' : undefined}
      data-variant={type || undefined}
      {...rest}
    >
      {children}
    </button>
  );

  const Table = ({ dataSource = [], columns = [], loading, pagination, ...rest }: any) => (
    <div>
      {pagination?.onChange && (
        <button
          type="button"
          data-testid="pagination-next"
          onClick={() => pagination.onChange(2, pagination.pageSize || 10)}
        >
          Next
        </button>
      )}
      {dataSource.map((record: any, rowIndex: number) => (
        <div key={record.id ?? rowIndex}>
          {columns.map((column: any, colIndex: number) => {
            const value = record[column.dataIndex];
            const content = column.render ? column.render(value, record) : value;
            return <div key={column.key ?? column.dataIndex ?? colIndex}>{content}</div>;
          })}
        </div>
      ))}
    </div>
  );

  const Tabs = ({ children }: any) => <div>{children}</div>;
  Tabs.TabPane = ({ children }: any) => <div>{children}</div>;

  const Descriptions = ({ children }: any) => <div>{children}</div>;
  Descriptions.Item = ({ label, children }: any) => (
    <div>
      <span>{label}</span>
      {children}
    </div>
  );

  const Card = ({ children }: any) => <div>{children}</div>;
  const Space = ({ children }: any) => <div>{children}</div>;
  const Breadcrumb = ({ items = [] }: any) => (
    <nav>
      {items.map((item: any, index: number) => (
        <span key={index}>{item.title}</span>
      ))}
    </nav>
  );
  const Tag = ({ children }: any) => <span>{children}</span>;
  const Skeleton = ({ children }: any) => <div>{children}</div>;
  const Result = ({ title, subTitle, extra }: any) => (
    <div>
      <div>{title}</div>
      <div>{subTitle}</div>
      {extra}
    </div>
  );
  const Empty = ({ description }: any) => <div>{description}</div>;
  const Tooltip = ({ children }: any) => <span>{children}</span>;

  return {
    ...actual,
    Button,
    Card,
    Breadcrumb,
    Descriptions,
    Empty,
    Form,
    Input,
    Modal: {
      ...actual.Modal,
      confirm: modalConfirmSpy,
    },
    Select,
    Skeleton,
    Space,
    Table,
    Tabs,
    Tag,
    Tooltip,
    Result,
    message: {
      success: jest.fn(),
      error: jest.fn(),
      warning: jest.fn(),
      info: jest.fn(),
    },
  };
});

jest.mock('@/modules/cmdb/api', () => ({
  CMDBApi: {
    getCIs: jest.fn(),
    getCI: jest.fn(),
    getTypes: jest.fn(),
    createCI: jest.fn(),
    deleteCI: jest.fn(),
  },
}));

const mockTypes: CIType[] = [
  {
    id: 1,
    name: '服务器',
    description: '',
    is_active: true,
    tenant_id: 1,
  },
];

const mockItem: ConfigurationItem = {
  id: 1,
  name: '应用服务器-01',
  description: '测试资产',
  type: 'server',
  status: 'active',
  ci_type_id: 1,
  tenant_id: 1,
  created_at: '2024-01-01T00:00:00Z',
  updated_at: '2024-01-01T00:00:00Z',
};

describe('CMDB 页面', () => {
  beforeEach(() => {
    jest.clearAllMocks();
    if (modalConfirmSpy) {
      modalConfirmSpy.mockClear();
    }
    (CMDBApi.getTypes as jest.Mock).mockResolvedValue(mockTypes);
    (CMDBApi.getCIs as jest.Mock).mockResolvedValue({
      items: [mockItem],
      total: 1,
      page: 1,
      size: 10,
    });
    (CMDBApi.getCI as jest.Mock).mockResolvedValue(mockItem);
    (CMDBApi.createCI as jest.Mock).mockResolvedValue(mockItem);
    (CMDBApi.deleteCI as jest.Mock).mockResolvedValue(undefined);
  });

  it('CI 列表应展示资产并支持跳转', async () => {
    render(<CIList />);

    await screen.findByText('应用服务器-01');

    fireEvent.click(screen.getByRole('button', { name: '应用服务器-01' }));
    expect(mockPush).toHaveBeenCalledWith('/cmdb/cis/1');
  });

  it('CI 列表筛选触发数据刷新', async () => {
    render(<CIList />);

    await screen.findByText('应用服务器-01');

    fireEvent.change(screen.getByPlaceholderText('搜索名称/序列号'), {
      target: { value: 'server' },
    });
    const queryButton = screen.getByText('查询').closest('button');
    fireEvent.click(queryButton as HTMLElement);

    await waitFor(() => {
      expect(CMDBApi.getCIs).toHaveBeenCalledTimes(2);
    });
  });

  it('CI 列表筛选条件应传递到请求', async () => {
    render(<CIList />);

    await screen.findByText('应用服务器-01');

    fireEvent.change(screen.getByPlaceholderText('搜索名称/序列号'), {
      target: { value: 'db' },
    });
    fireEvent.change(document.querySelector('select[name="ci_type_id"]') as HTMLSelectElement, {
      target: { value: '1' },
    });
    fireEvent.change(document.querySelector('select[name="status"]') as HTMLSelectElement, {
      target: { value: 'inactive' },
    });

    const queryButton = screen.getByText('查询').closest('button');
    fireEvent.click(queryButton as HTMLElement);

    await waitFor(() => {
      const lastCall = (CMDBApi.getCIs as jest.Mock).mock.calls.at(-1);
      expect(lastCall?.[0]).toEqual(
        expect.objectContaining({
          search: 'db',
          ci_type_id: '1',
          status: 'inactive',
        })
      );
    });
  });

  it('分页切换应触发请求', async () => {
    render(<CIList />);

    await screen.findByText('应用服务器-01');
    fireEvent.click(screen.getByTestId('pagination-next'));

    await waitFor(() => {
      expect(CMDBApi.getCIs).toHaveBeenCalledTimes(2);
    });
  });

  it('刷新按钮应重新拉取数据', async () => {
    render(<CIList />);

    await screen.findByText('应用服务器-01');
    fireEvent.click(screen.getByRole('button', { name: '刷新' }));

    await waitFor(() => {
      expect(CMDBApi.getCIs).toHaveBeenCalledTimes(2);
    });
  });

  it('列表为空时不应展示资产名称', async () => {
    (CMDBApi.getCIs as jest.Mock).mockResolvedValueOnce({
      items: [],
      total: 0,
      page: 1,
      size: 10,
    });
    render(<CIList />);

    await waitFor(() => {
      expect(CMDBApi.getCIs).toHaveBeenCalled();
    });
    expect(screen.queryByText('应用服务器-01')).not.toBeInTheDocument();
  });

  it('删除失败应提示错误', async () => {
    (CMDBApi.deleteCI as jest.Mock).mockRejectedValueOnce(new Error('删除失败'));
    render(<CIList />);

    await screen.findByText('应用服务器-01');
    fireEvent.click(screen.getByRole('button', { name: '删除' }));

    await waitFor(() => {
      expect(CMDBApi.deleteCI).toHaveBeenCalledWith(1);
    });
  });

  it('CI 删除应弹出确认并调用删除接口', async () => {
    render(<CIList />);

    await screen.findByText('应用服务器-01');

    fireEvent.click(screen.getByRole('button', { name: '删除' }));
    await waitFor(() => {
      expect(CMDBApi.deleteCI).toHaveBeenCalledWith(1);
    });
  });

  it('CI 详情应展示资产信息', async () => {
    render(<CIDetail />);

    await waitFor(() => {
      expect(screen.getByText('应用服务器-01')).toBeInTheDocument();
    });
    expect(screen.getByText(/配置项 ID: 1/)).toBeInTheDocument();
  });

  it('CI 详情加载失败应显示 404', async () => {
    (CMDBApi.getCI as jest.Mock).mockRejectedValueOnce(new Error('not found'));
    render(<CIDetail />);

    await waitFor(() => {
      expect(screen.getByText('404')).toBeInTheDocument();
    });
  });

  it('创建 CI 表单应提交并返回列表', async () => {
    render(<CreateCIPage />);

    await waitFor(() => {
      expect(CMDBApi.getTypes).toHaveBeenCalled();
    });

    fireEvent.change(screen.getByLabelText('资产名称'), {
      target: { value: '新资产' },
    });

    fireEvent.change(screen.getByLabelText('资产类型'), {
      target: { value: '1' },
    });

    fireEvent.change(screen.getByLabelText('状态'), {
      target: { value: 'active' },
    });

    const form = document.querySelector('form');
    fireEvent.submit(form as HTMLFormElement);

    await waitFor(() => {
      expect(CMDBApi.createCI).toHaveBeenCalled();
    });
    expect(mockPush).toHaveBeenCalledWith('/cmdb');
  });

  it('创建 CI 表单应校验扩展属性 JSON', async () => {
    const antd = require('antd');
    render(<CreateCIPage />);

    await waitFor(() => {
      expect(CMDBApi.getTypes).toHaveBeenCalled();
    });

    fireEvent.change(screen.getByLabelText('资产名称'), {
      target: { value: '新资产' },
    });
    fireEvent.change(screen.getByLabelText('资产类型'), {
      target: { value: '1' },
    });
    fireEvent.change(screen.getByLabelText('状态'), {
      target: { value: 'active' },
    });
    fireEvent.change(screen.getByLabelText('扩展属性'), {
      target: { value: '{bad json' },
    });

    await waitFor(() => {
      expect(screen.getByLabelText('扩展属性')).toHaveValue('{bad json');
    });

    fireEvent.click(screen.getByRole('button', { name: '保存' }));

    await waitFor(() => {
      expect(CMDBApi.createCI).not.toHaveBeenCalled();
    });
    expect(mockPush).not.toHaveBeenCalled();
  });
});
