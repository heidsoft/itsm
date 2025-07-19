import React from "react";
import { FormInput } from "../FormInput";
import { FormTextarea } from "../FormTextarea";

interface FormFieldProps {
  type:
    | "input"
    | "textarea"
    | "select"
    | "checkbox"
    | "radio"
    | "date"
    | "file";
  label: string;
  name: string;
  value: unknown;
  onChange: (value: unknown) => void;
  options?: Array<{ label: string; value: unknown }>;
  validation?: {
    required?: boolean;
    pattern?: RegExp;
    minLength?: number;
    maxLength?: number;
    custom?: (value: unknown) => string | null;
  };
  placeholder?: string;
  disabled?: boolean;
  className?: string;
}

export const FormField: React.FC<FormFieldProps> = ({
  type,
  label,
  name,
  value,
  onChange,
  options,
  validation,
  placeholder,
  disabled,
  className,
}) => {
  const [error, setError] = React.useState<string | null>(null);

  const validateField = (val: unknown) => {
    if (!validation) return null;

    if (validation.required && (!val || val.toString().trim() === "")) {
      return `${label}是必填项`;
    }

    if (validation.pattern && !validation.pattern.test(val)) {
      return `${label}格式不正确`;
    }

    if (validation.custom) {
      return validation.custom(val);
    }

    return null;
  };

  const handleChange = (newValue: unknown) => {
    const validationError = validateField(newValue);
    setError(validationError);
    onChange(newValue);
  };

  const renderField = () => {
    switch (type) {
      case "input":
        return (
          <FormInput
            label={label}
            value={value}
            onChange={(e) => handleChange(e.target.value)}
            placeholder={placeholder}
            disabled={disabled}
            className={className}
          />
        );
      case "textarea":
        return (
          <FormTextarea
            label={label}
            value={value}
            onChange={(e) => handleChange(e.target.value)}
            placeholder={placeholder}
            disabled={disabled}
            className={className}
          />
        );
      case "select":
        return (
          <div className="space-y-2">
            <label className="block text-sm font-medium text-gray-700">
              {label}
            </label>
            <select
              value={value}
              onChange={(e) => handleChange(e.target.value)}
              disabled={disabled}
              className={`w-full px-4 py-3 border border-gray-200 rounded-xl focus:ring-2 focus:ring-blue-500 focus:border-transparent transition-all duration-200 bg-gray-50 focus:bg-white hover:bg-white ${
                className || ""
              }`}
            >
              <option value="">{placeholder || "请选择..."}</option>
              {options?.map((option) => (
                <option key={option.value} value={option.value}>
                  {option.label}
                </option>
              ))}
            </select>
          </div>
        );
      // ... 其他类型
      default:
        return null;
    }
  };

  return (
    <div className="space-y-1">
      {renderField()}
      {error && (
        <p className="text-sm text-red-600 flex items-center">
          <span className="mr-1">⚠️</span>
          {error}
        </p>
      )}
    </div>
  );
};
