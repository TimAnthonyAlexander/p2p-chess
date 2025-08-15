import { ChangeEvent, forwardRef, SelectHTMLAttributes } from 'react';

interface Option {
  value: string;
  label: string;
}

interface SelectProps extends SelectHTMLAttributes<HTMLSelectElement> {
  label?: string;
  options: Option[];
  error?: string;
  fullWidth?: boolean;
}

const Select = forwardRef<HTMLSelectElement, SelectProps>(
  ({ label, options, error, fullWidth = false, className = '', ...props }, ref) => {
    const handleChange = (e: ChangeEvent<HTMLSelectElement>) => {
      if (props.onChange) {
        props.onChange(e);
      }
    };

    const selectClasses = [
      'rounded-md border px-3 py-2 text-gray-900 shadow-sm focus:outline-none focus:ring-2 focus:ring-blue-500',
      error ? 'border-red-500' : 'border-gray-300',
      fullWidth ? 'w-full' : '',
      props.disabled ? 'bg-gray-100 cursor-not-allowed' : '',
      className
    ].join(' ');

    const labelClasses = 'block text-sm font-medium text-gray-700 mb-1';

    return (
      <div className={fullWidth ? 'w-full' : ''}>
        {label && (
          <label htmlFor={props.id} className={labelClasses}>
            {label}
          </label>
        )}
        <select
          ref={ref}
          {...props}
          className={selectClasses}
          onChange={handleChange}
        >
          {options.map((option) => (
            <option key={option.value} value={option.value}>
              {option.label}
            </option>
          ))}
        </select>
        {error && <p className="mt-1 text-sm text-red-500">{error}</p>}
      </div>
    );
  }
);

Select.displayName = 'Select';

export default Select;
