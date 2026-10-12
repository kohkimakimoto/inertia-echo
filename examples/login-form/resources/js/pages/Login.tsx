import React, { useState } from 'react';
import { Head, useForm } from '@inertiajs/react';

type LoginProps = {
  errors?: {
    email?: string;
    password?: string;
  };
};

export default function Login({ errors = {} }: LoginProps) {
  const { data, setData, post, processing } = useForm({
    email: '',
    password: '',
  });

  const handleSubmit = (e: React.FormEvent) => {
    e.preventDefault();
    post('/login');
  };

  return (
    <>
      <Head>
        <title>Login</title>
      </Head>
      <div style={containerStyle}>
        <div style={cardStyle}>
          <h1 style={titleStyle}>Login</h1>
          <form onSubmit={handleSubmit} style={formStyle}>
            <div style={fieldGroupStyle}>
              <label htmlFor="email" style={labelStyle}>
                Email
              </label>
              <input
                id="email"
                type="email"
                value={data.email}
                onChange={(e) => setData('email', e.target.value)}
                style={{
                  ...inputStyle,
                  ...(errors.email ? errorInputStyle : {}),
                }}
                required
              />
              {errors.email && (
                <span style={errorTextStyle}>{errors.email}</span>
              )}
            </div>

            <div style={fieldGroupStyle}>
              <label htmlFor="password" style={labelStyle}>
                Password
              </label>
              <input
                id="password"
                type="password"
                value={data.password}
                onChange={(e) => setData('password', e.target.value)}
                style={{
                  ...inputStyle,
                  ...(errors.password ? errorInputStyle : {}),
                }}
                required
              />
              {errors.password && (
                <span style={errorTextStyle}>{errors.password}</span>
              )}
            </div>

            <button
              type="submit"
              disabled={processing}
              style={{
                ...buttonStyle,
                ...(processing ? buttonDisabledStyle : {}),
              }}
            >
              {processing ? 'Logging in...' : 'Login'}
            </button>
          </form>
        </div>
      </div>
    </>
  );
}

// Styles
const containerStyle: React.CSSProperties = {
  display: 'flex',
  justifyContent: 'center',
  alignItems: 'center',
  minHeight: '100vh',
  backgroundColor: '#f5f5f5',
  fontFamily: '-apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, sans-serif',
};

const cardStyle: React.CSSProperties = {
  backgroundColor: 'white',
  padding: '2rem',
  borderRadius: '8px',
  boxShadow: '0 4px 6px rgba(0, 0, 0, 0.1)',
  width: '100%',
  maxWidth: '400px',
};

const titleStyle: React.CSSProperties = {
  fontSize: '2rem',
  fontWeight: 'bold',
  textAlign: 'center',
  marginBottom: '1.5rem',
  color: '#333',
};

const formStyle: React.CSSProperties = {
  display: 'flex',
  flexDirection: 'column',
  gap: '1rem',
};

const fieldGroupStyle: React.CSSProperties = {
  display: 'flex',
  flexDirection: 'column',
};

const labelStyle: React.CSSProperties = {
  fontSize: '0.875rem',
  fontWeight: '600',
  color: '#374151',
  marginBottom: '0.5rem',
};

const inputStyle: React.CSSProperties = {
  padding: '0.75rem',
  border: '1px solid #d1d5db',
  borderRadius: '6px',
  fontSize: '1rem',
  transition: 'border-color 0.15s ease-in-out, box-shadow 0.15s ease-in-out',
  outline: 'none',
};

const errorInputStyle: React.CSSProperties = {
  borderColor: '#ef4444',
  boxShadow: '0 0 0 3px rgba(239, 68, 68, 0.1)',
};

const errorTextStyle: React.CSSProperties = {
  color: '#ef4444',
  fontSize: '0.875rem',
  marginTop: '0.25rem',
};

const buttonStyle: React.CSSProperties = {
  backgroundColor: '#3b82f6',
  color: 'white',
  padding: '0.75rem 1rem',
  border: 'none',
  borderRadius: '6px',
  fontSize: '1rem',
  fontWeight: '600',
  cursor: 'pointer',
  transition: 'background-color 0.15s ease-in-out',
  marginTop: '0.5rem',
};

const buttonDisabledStyle: React.CSSProperties = {
  backgroundColor: '#9ca3af',
  cursor: 'not-allowed',
};
