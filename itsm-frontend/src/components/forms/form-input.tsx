'use client';

import React, { forwardRef } from 'react';
import { FormInputProps } from './types';
import { FormField } from './form-field';

const Input = forwardRef<HTMLInputElement, Omit<FormInputProps, 'label' | 'error' | 'help'>>(
  ({
    type = 'text',
    value = '',
    placeholder,
    disabled = false,
    readonly = false,
    maxLength,
    minLength,
    pattern,
    autoComplete,
    autoFocus = false,
    prefix,
    suffix,
    onChange,
    onBlur,
    onFocus,
    className = '',
    ...props
  }, ref) => {
    const handleChange = (e: React.ChangeEvent<HTMLInputElement>) => {
      onChange?.(e.target.value);
    };

    const inputClasses = [
      'block w-full rounded-md border-gray-300 shadow-sm',
      'focus:border-blue-500 focus:ring-blue-500',
      'disabled:bg-gray-50 disabled:text-gray-500 disabled:cursor-not-allowed',
      'readonly:bg-gray-50 readonly:cursor-default',
      prefix ? 'pl-10' : 'px-3',
      suffix ? 'pr-10' : 'px-3',
      'py-2',
      className
    ].filter(Boolean).join(' ');

    return (
      <div className="relative">
        {prefix && (
          <div className="absolute inset-y-0 left-0 pl-3 flex items-center pointer-events-none">
            <span className="text-gray-500 sm:text-sm">{prefix}</span>
          </div>
        )}
        
        <input
          ref={ref}
          type={type}
          value={value}
          placeholder={placeholder}
          disabled={disabled}
          readOnly={readonly}
          maxLength={maxLength}
          minLength={minLength}
          pattern={pattern}
          autoComplete={autoComplete}
          autoFocus={autoFocus}
          onChange={handleChange}
          onBlur={onBlur}
          onFocus={onFocus}
          className={inputClasses}
          {...props}
        />
        
        {suffix && (
          <div className="absolute inset-y-0 right-0 pr-3 flex items-center pointer-events-none">
            <span className="text-gray-500 sm:text-sm">{suffix}</span>
          </div>
        )}
      </div>
    );
  }
);

Input.displayName = 'Input';

export const FormInput: React.FC<FormInputProps> = (props) => {
  const { label, error, help, ...inputProps } = props;

  if (label || error || help) {
    return (
      <FormField
        name={props.name}
        label={label}
        required={props.required}
        error={error}
        help={help}
      >
        <Input {...inputProps} />
      </FormField>
    );
  }

  return <Input {...inputProps} />;
};