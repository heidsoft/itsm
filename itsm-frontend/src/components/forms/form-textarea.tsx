'use client';

import React, { forwardRef, useEffect, useRef } from 'react';
import { FormTextareaProps } from './types';
import { FormField } from './form-field';

const Textarea = forwardRef<HTMLTextAreaElement, Omit<FormTextareaProps, 'label' | 'error' | 'help'>>(
  ({
    value = '',
    placeholder,
    disabled = false,
    readonly = false,
    rows = 4,
    cols,
    maxLength,
    minLength,
    resize = 'vertical',
    autoResize = false,
    onChange,
    onBlur,
    onFocus,
    className = '',
    ...props
  }, ref) => {
    const textareaRef = useRef<HTMLTextAreaElement>(null);
    const combinedRef = ref || textareaRef;

    const handleChange = (e: React.ChangeEvent<HTMLTextAreaElement>) => {
      onChange?.(e.target.value);
    };

    // 自动调整高度
    useEffect(() => {
      if (autoResize && combinedRef && 'current' in combinedRef && combinedRef.current) {
        const textarea = combinedRef.current;
        textarea.style.height = 'auto';
        textarea.style.height = `${textarea.scrollHeight}px`;
      }
    }, [value, autoResize, combinedRef]);

    const textareaClasses = [
      'block w-full rounded-md border-gray-300 shadow-sm',
      'focus:border-blue-500 focus:ring-blue-500',
      'disabled:bg-gray-50 disabled:text-gray-500 disabled:cursor-not-allowed',
      'readonly:bg-gray-50 readonly:cursor-default',
      'px-3 py-2',
      resize === 'none' ? 'resize-none' : 
      resize === 'horizontal' ? 'resize-x' :
      resize === 'vertical' ? 'resize-y' : 'resize',
      className
    ].filter(Boolean).join(' ');

    const currentLength = typeof value === 'string' ? value.length : 0;
    const showCounter = maxLength && maxLength > 0;

    return (
      <div className="relative">
        <textarea
          ref={combinedRef}
          value={value}
          placeholder={placeholder}
          disabled={disabled}
          readOnly={readonly}
          rows={rows}
          cols={cols}
          maxLength={maxLength}
          minLength={minLength}
          onChange={handleChange}
          onBlur={onBlur}
          onFocus={onFocus}
          className={textareaClasses}
          {...props}
        />
        
        {showCounter && (
          <div className="absolute bottom-2 right-2 text-xs text-gray-400 bg-white px-1">
            {currentLength}/{maxLength}
          </div>
        )}
      </div>
    );
  }
);

Textarea.displayName = 'Textarea';

export const FormTextarea: React.FC<FormTextareaProps> = (props) => {
  const { label, error, help, ...textareaProps } = props;

  if (label || error || help) {
    return (
      <FormField
        name={props.name}
        label={label}
        required={props.required}
        error={error}
        help={help}
      >
        <Textarea {...textareaProps} />
      </FormField>
    );
  }

  return <Textarea {...textareaProps} />;
};