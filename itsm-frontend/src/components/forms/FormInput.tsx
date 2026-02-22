import React, { useState } from "react";
import { Form } from "antd";

interface FormInputProps extends React.InputHTMLAttributes<HTMLInputElement> {
  label: string;
  error?: string;
  help?: string;
  required?: boolean;
  id?: string;
}

export const FormInput: React.FC<FormInputProps> = ({
  label,
  error,
  help,
  required = false,
  id,
  className = "",
  ...props
}) => {
  const [focused, setFocused] = useState(false);

  return (
    <div className="space-y-2">
      <label
        htmlFor={id}
        className="block text-sm font-medium text-gray-700 mb-2"
      >
        {label}
        {required && <span className="text-red-500 ml-1">*</span>}
      </label>
      <input
        id={id}
        aria-label={label}
        aria-invalid={!!error}
        aria-describedby={`${id}-help ${id}-error`}
        className={`w-full px-4 py-3 border rounded-xl transition-all duration-200 placeholder-gray-400 focus:outline-none ${
          className || ""
        } ${
          error
            ? "border-red-500 focus:ring-2 focus:ring-red-200 focus:border-red-500 bg-red-50"
            : "border-gray-200 focus:ring-2 focus:ring-blue-500 focus:border-transparent bg-gray-50 focus:bg-white hover:bg-white"
        }`}
        onFocus={() => setFocused(true)}
        onBlur={() => setFocused(false)}
        {...props}
      />
      {help && !error && (
        <p id={`${id}-help`} className="text-xs text-gray-500">
          {help}
        </p>
      )}
      {error && (
        <p id={`${id}-error`} className="text-xs text-red-500 mt-1 flex items-center gap-1">
          <svg
            className="w-3 h-3"
            fill="currentColor"
            viewBox="0 0 20 20"
          >
            <path
              fillRule="evenodd"
              d="M18 10a8 8 0 11-16 0 8 8 0 0116 0zm-7 4a1 1 0 11-2 0 1 1 0 012 0zm-1-9a1 1 0 00-1 1v4a1 1 0 102 0V6a1 1 0 00-1-1z"
              clipRule="evenodd"
            />
          </svg>
          {error}
        </p>
      )}
    </div>
  );
};
