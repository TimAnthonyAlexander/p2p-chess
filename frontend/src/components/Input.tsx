import { ChangeEvent, InputHTMLAttributes, forwardRef } from 'react';

interface InputProps extends InputHTMLAttributes<HTMLInputElement> {
  label?: string;
  error?: string;
  fullWidth?: boolean;
}

const Input = forwardRef<HTMLInputElement, InputProps>(
  ({ label, error, fullWidth = false, className = '', ...props }, ref) => {
    const handleChange = (e: ChangeEvent<HTMLInputElement>) => {
      if (props.onChange) {
        props.onChange(e);
      }
    };

    const inputClasses = [
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
        <input
          ref={ref}
          {...props}
          className={inputClasses}
          onChange={handleChange}
        />
        {error && <p className="mt-1 text-sm text-red-500">{error}</p>}
      </div>
    );
  }
);

Input.displayName = 'Input';

export default Input;
