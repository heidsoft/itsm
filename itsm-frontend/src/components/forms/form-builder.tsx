'use client';

import React from 'react';
import { FormBuilderProps, FormFieldConfig } from './types';
import { FormInput } from './form-input';
import { FormTextarea } from './form-textarea';
import { FormSelect } from './form-select';
import { FormCheckbox } from './form-checkbox';
import { FormRadioGroup } from './form-radio';
import { FormDatePicker } from './form-date-picker';

export const FormBuilder: React.FC<FormBuilderProps> = ({
  fields,
  values = {},
  errors = {},
  onChange,
  onSubmit,
  loading = false,
  submitText = '提交',
  resetText = '重置',
  showReset = true,
  layout = 'vertical',
  className = ''
}) => {
  const handleFieldChange = (name: string, value: unknown) => {
    onChange?.(name, value);
  };

  const handleSubmit = (e: React.FormEvent) => {
    e.preventDefault();
    onSubmit?.(values);
  };

  const handleReset = () => {
    const resetValues: Record<string, unknown> = {};
    fields.forEach(field => {
      resetValues[field.name] = field.defaultValue || '';
    });
    
    fields.forEach(field => {
      onChange?.(field.name, resetValues[field.name]);
    });
  };

  const renderField = (field: FormFieldConfig) => {
    const commonProps = {
      name: field.name,
      label: field.label,
      placeholder: field.placeholder,
      required: field.required,
      disabled: field.disabled || loading,
      error: errors[field.name],
      help: field.help,
      onChange: (value: unknown) => handleFieldChange(field.name, value),
    };

    switch (field.type) {
      case 'text':
      case 'email':
      case 'password':
      case 'number':
      case 'tel':
      case 'url':
        return (
          <FormInput
            key={field.name}
            {...commonProps}
            type={field.type}
            value={values[field.name] as string}
            maxLength={field.maxLength}
            minLength={field.minLength}
            pattern={field.pattern}
          />
        );

      case 'textarea':
        return (
          <FormTextarea
            key={field.name}
            {...commonProps}
            value={values[field.name] as string}
            rows={field.rows}
            maxLength={field.maxLength}
            minLength={field.minLength}
            autoResize={field.autoResize}
          />
        );

      case 'select':
        return (
          <FormSelect
            key={field.name}
            {...commonProps}
            value={values[field.name] as string | number | string[] | number[]}
            options={field.options || []}
            multiple={field.multiple}
            searchable={field.searchable}
            clearable={field.clearable}
          />
        );

      case 'checkbox':
        return (
          <FormCheckbox
            key={field.name}
            {...commonProps}
            checked={Boolean(values[field.name])}
            value={values[field.name] as string | number}
          >
            {field.label}
          </FormCheckbox>
        );

      case 'radio':
        return (
          <FormRadioGroup
            key={field.name}
            {...commonProps}
            value={values[field.name] as string | number}
            options={field.options || []}
            direction={field.direction}
          />
        );

      case 'date':
        return (
          <FormDatePicker
            key={field.name}
            {...commonProps}
            value={values[field.name] as string | Date}
            showTime={field.showTime}
            showToday={field.showToday}
            disabledDate={field.disabledDate}
          />
        );

      default:
        console.warn(`Unknown field type: ${field.type}`);
        return null;
    }
  };

  return (
    <form onSubmit={handleSubmit} className={`space-y-6 ${className}`}>
      <div className={`${layout === 'horizontal' ? 'grid grid-cols-2 gap-6' : 'space-y-4'}`}>
        {fields.map(renderField)}
      </div>

      <div className="flex justify-end space-x-4 pt-6 border-t border-gray-200">
        {showReset && (
          <button
            type="button"
            onClick={handleReset}
            disabled={loading}
            className="px-4 py-2 text-sm font-medium text-gray-700 bg-white border border-gray-300 rounded-md hover:bg-gray-50 focus:outline-none focus:ring-2 focus:ring-offset-2 focus:ring-blue-500 disabled:opacity-50 disabled:cursor-not-allowed"
          >
            {resetText}
          </button>
        )}
        
        <button
          type="submit"
          disabled={loading}
          className="px-4 py-2 text-sm font-medium text-white bg-blue-600 border border-transparent rounded-md hover:bg-blue-700 focus:outline-none focus:ring-2 focus:ring-offset-2 focus:ring-blue-500 disabled:opacity-50 disabled:cursor-not-allowed"
        >
          {loading ? '提交中...' : submitText}
        </button>
      </div>
    </form>
  );
};