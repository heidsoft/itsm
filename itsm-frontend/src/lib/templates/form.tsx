"use client";

import React, { useState, ReactNode } from "react";
import { Form, Input, Button, Select, DatePicker, InputNumber, Space, Modal } from "antd";
import type { Rule } from "antd/es/form";
import type { FormInstance } from "antd/es/form";

interface FormConfig<T> {
  initialValues?: Partial<T>;
  rules?: Partial<Record<keyof T, Rule[]>>;
  layout?: "horizontal" | "vertical" | "inline";
  onSubmit?: (values: T) => Promise<void>;
  onSuccess?: () => void;
}

export function createForm<T extends Record<string, unknown>>({
  initialValues,
  rules = {},
  layout = "vertical",
  onSubmit,
  onSuccess,
}: FormConfig<T>) {
  return function BaseForm({
    children,
    submitText = "提交",
    showReset = true,
    disabled = false,
  }: {
    children: (form: FormInstance<T>) => ReactNode;
    submitText?: string;
    showReset?: boolean;
    disabled?: boolean;
  }) {
    const [form] = Form.useForm<T>();
    const [loading, setLoading] = useState(false);

    const handleSubmit = async () => {
      try {
        setLoading(true);
        const values = await form.validateFields();
        if (onSubmit) await onSubmit(values);
        onSuccess?.();
      } catch (error) {
        if (error instanceof Error && (error as { errorFields?: unknown[] }).errorFields) return;
        console.error("表单提交错误:", error);
      } finally {
        setLoading(false);
      }
    };

    return (
      <Form form={form} layout={layout} disabled={loading || disabled} initialValues={initialValues as T}>
        {children(form)}
        <Form.Item style={{ marginTop: 24 }}>
          <Space>
            <Button type="primary" onClick={handleSubmit} loading={loading}>{submitText}</Button>
            {showReset && <Button onClick={() => form.resetFields()}>重置</Button>}
          </Space>
        </Form.Item>
      </Form>
    );
  };
}

interface SearchFormConfig {
  onSearch: (values: Record<string, unknown>) => void;
  onReset: () => void;
}

export function createSearchForm(config: SearchFormConfig) {
  return function SearchForm({
    children,
    layout = "inline",
  }: {
    children: (form: FormInstance) => ReactNode;
    layout?: "horizontal" | "vertical" | "inline";
  }) {
    const [form] = Form.useForm();
    const [loading, setLoading] = useState(false);

    const handleSearch = async () => {
      try {
        setLoading(true);
        const values = await form.validateFields();
        config.onSearch(values);
      } finally {
        setLoading(false);
      }
    };

    return (
      <Form form={form} layout={layout} style={{ marginBottom: 16 }}>
        <Space wrap>
          {children(form)}
          <Button type="primary" onClick={handleSearch} loading={loading}>搜索</Button>
          <Button onClick={() => { form.resetFields(); config.onReset(); }}>重置</Button>
        </Space>
      </Form>
    );
  };
}

interface FieldProps {
  name: string;
  label: string;
  required?: boolean;
  placeholder?: string;
  disabled?: boolean;
  rules?: Rule[];
}

export function TextField(props: FieldProps) {
  return (
    <Form.Item name={props.name} label={props.label} required={props.required} rules={props.rules}>
      <Input placeholder={props.placeholder} disabled={props.disabled} />
    </Form.Item>
  );
}

export function SelectField({
  name, label, required, placeholder, disabled, options, showSearch, rules,
}: FieldProps & { options: { label: string; value: string | number }[]; showSearch?: boolean }) {
  return (
    <Form.Item name={name} label={label} required={required} rules={rules}>
      <Select
        placeholder={placeholder}
        disabled={disabled}
        options={options}
        showSearch={showSearch}
        filterOption={(input, option) =>
          (option?.label ?? "").toLowerCase().includes(input.toLowerCase())
        }
      />
    </Form.Item>
  );
}

interface UseModalFormProps<T> {
  title: string;
  width?: number | string;
  onSubmit: (values: T) => Promise<void>;
  onSuccess?: () => void;
}

export function useModalForm<T extends Record<string, unknown>>(config: UseModalFormProps<T>) {
  const [open, setOpen] = useState(false);
  const [form] = Form.useForm<T>();
  const [loading, setLoading] = useState(false);

  const show = () => setOpen(true);
  const hide = () => setOpen(false);

  const handleSubmit = async () => {
    try {
      setLoading(true);
      const values = await form.validateFields();
      await config.onSubmit(values);
      hide();
      config.onSuccess?.();
    } catch (error) {
      if (error instanceof Error && (error as { errorFields?: unknown[] }).errorFields) return;
    } finally {
      setLoading(false);
    }
  };

  const ModalFormComponent = ({ children }: { children: (form: FormInstance<T>) => ReactNode }) => (
    <Modal
      title={config.title}
      open={open}
      onCancel={hide}
      onOk={handleSubmit}
      confirmLoading={loading}
      width={config.width}
    >
      <Form form={form}>{children(form)}</Form>
    </Modal>
  );

  return { open, show, hide, form, loading, ModalFormComponent };
}
