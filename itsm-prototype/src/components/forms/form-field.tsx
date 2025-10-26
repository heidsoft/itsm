'use client';

import React from 'react';
import { FormFieldProps } from './types';

export const FormField: React.FC<FormFieldProps> = ({
  name,
  label,
  required = false,
  error,
  help,
  children,
  layout = 'vertical',
  labelWidth = '120px',
  className = ''
}) => {
  const fieldId = `field-${name}`;
  const hasError = Boolean(error);
  const errorMessages = Array.isArray(error) ? error : error ? [error] : [];

  const labelElement = label && (
    <label
      htmlFor={fieldId}
      className={`block text-sm font-medium text-gray-700 ${
        layout === 'horizontal' ? 'flex-shrink-0' : 'mb-1'
      }`}
      style={layout === 'horizontal' ? { width: labelWidth } : undefined}
    >
      {label}
      {required && <span className="text-red-500 ml-1">*</span>}
    </label>
  );

  const fieldContent = (
    <div className="flex-1">
      <div className="relative">
        {React.cloneElement(children as React.ReactElement, {
          id: fieldId,
          name,
          className: `${hasError ? 'border-red-300 focus:border-red-500 focus:ring-red-500' : ''}`
        })}
      </div>
      
      {/* 错误信息 */}
      {hasError && (
        <div className="mt-1">
          {errorMessages.map((msg, index) => (
            <p key={index} className="text-sm text-red-600">
              {msg}
            </p>
          ))}
        </div>
      )}
      
      {/* 帮助信息 */}
      {help && !hasError && (
        <p className="mt-1 text-sm text-gray-500">{help}</p>
      )}
    </div>
  );

  return (
    <div className={`form-field ${className}`}>
      {layout === 'horizontal' ? (
        <div className="flex items-start space-x-4">
          {labelElement}
          {fieldContent}
        </div>
      ) : (
        <div>
          {labelElement}
          {fieldContent}
        </div>
      )}
    </div>
  );
};