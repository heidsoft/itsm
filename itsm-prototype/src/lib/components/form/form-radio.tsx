'use client';

import React, { forwardRef } from 'react';
import { FormRadioProps, FormRadioGroupProps, RadioOption } from './types';
import { FormField } from './form-field';

const Radio = forwardRef<HTMLInputElement, Omit<FormRadioProps, 'label' | 'error' | 'help'>>(
  ({ checked, value, disabled, onChange, onBlur, onFocus, children, className = '', ...props }, ref) => {
    const handleChange = (e: React.ChangeEvent<HTMLInputElement>) => {
      if (e.target.checked) {
        onChange?.(value);
      }
    };

    return (
      <label className={`inline-flex items-center cursor-pointer ${disabled ? 'opacity-50 cursor-not-allowed' : ''} ${className}`}>
        <div className="relative">
          <input
            ref={ref}
            type="radio"
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
              w-4 h-4 border-2 rounded-full transition-colors duration-200
              ${checked 
                ? 'bg-blue-600 border-blue-600' 
                : 'bg-white border-gray-300 hover:border-gray-400'
              }
              ${disabled ? 'opacity-50' : ''}
            `}
          >
            {checked && (
              <div className="w-2 h-2 bg-white rounded-full absolute top-1/2 left-1/2 transform -translate-x-1/2 -translate-y-1/2" />
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

Radio.displayName = 'Radio';

const RadioGroup: React.FC<Omit<FormRadioGroupProps, 'label' | 'error' | 'help'>> = ({
  name,
  value,
  options = [],
  direction = 'vertical',
  disabled = false,
  onChange,
  className = ''
}) => {
  return (
    <div className={`${direction === 'horizontal' ? 'flex flex-wrap gap-4' : 'space-y-2'} ${className}`}>
      {options.map((option: RadioOption) => (
        <Radio
          key={option.value}
          name={name}
          value={option.value}
          checked={value === option.value}
          disabled={disabled || option.disabled}
          onChange={onChange}
        >
          {option.label}
        </Radio>
      ))}
    </div>
  );
};

export const FormRadio: React.FC<FormRadioProps> = (props) => {
  const { label, error, help, ...radioProps } = props;

  if (!label) {
    return <Radio {...radioProps} />;
  }

  return (
    <FormField label={label} error={error} help={help} name={props.name}>
      <Radio {...radioProps} />
    </FormField>
  );
};

export const FormRadioGroup: React.FC<FormRadioGroupProps> = (props) => {
  const { label, error, help, ...radioGroupProps } = props;

  return (
    <FormField label={label} error={error} help={help} name={props.name}>
      <RadioGroup {...radioGroupProps} />
    </FormField>
  );
};

export { Radio, RadioGroup };