'use client';

import React, { forwardRef } from 'react';
import { FormCheckboxProps } from './types';
import { FormField } from './form-field';

const Checkbox = forwardRef<HTMLInputElement, Omit<FormCheckboxProps, 'label' | 'error' | 'help'>>(
  ({ checked, value, indeterminate, disabled, onChange, onBlur, onFocus, children, className = '', ...props }, ref) => {
    const handleChange = (e: React.ChangeEvent<HTMLInputElement>) => {
      onChange?.(e.target.checked);
    };

    return (
      <label className={`inline-flex items-center cursor-pointer ${disabled ? 'opacity-50 cursor-not-allowed' : ''} ${className}`}>
        <div className="relative">
          <input
            ref={ref}
            type="checkbox"
            checked={checked}
            value={value}
            disabled={disabled}
            onChange={handleChange}
            onBlur={onBlur}
            onFocus={onFocus}
            className="sr-only"
            {...props}
          />
          <div
            className={`
              w-4 h-4 border-2 rounded transition-colors duration-200
              ${checked 
                ? 'bg-blue-600 border-blue-600' 
                : 'bg-white border-gray-300 hover:border-gray-400'
              }
              ${disabled ? 'opacity-50' : ''}
              ${indeterminate ? 'bg-blue-600 border-blue-600' : ''}
            `}
          >
            {checked && !indeterminate && (
              <svg className="w-3 h-3 text-white" fill="currentColor" viewBox="0 0 20 20">
                <path
                  fillRule="evenodd"
                  d="M16.707 5.293a1 1 0 010 1.414l-8 8a1 1 0 01-1.414 0l-4-4a1 1 0 011.414-1.414L8 12.586l7.293-7.293a1 1 0 011.414 0z"
                  clipRule="evenodd"
                />
              </svg>
            )}
            {indeterminate && (
              <svg className="w-3 h-3 text-white" fill="currentColor" viewBox="0 0 20 20">
                <path fillRule="evenodd" d="M4 10a1 1 0 011-1h10a1 1 0 110 2H5a1 1 0 01-1-1z" clipRule="evenodd" />
              </svg>
            )}
          </div>
        </div>
        {children && (
          <span className={`ml-2 text-sm ${disabled ? 'text-gray-400' : 'text-gray-700'}`}>
            {children}
          </span>
        )}
      </label>
    );
  }
);

Checkbox.displayName = 'Checkbox';

export const FormCheckbox: React.FC<FormCheckboxProps> = (props) => {
  const { label, error, help, ...checkboxProps } = props;

  if (!label) {
    return <Checkbox {...checkboxProps} />;
  }

  return (
    <FormField label={label} error={error} help={help} name={props.name}>
      <Checkbox {...checkboxProps} />
    </FormField>
  );
};

export { Checkbox };