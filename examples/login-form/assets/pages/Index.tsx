import React from 'react';
import { Head, Link } from '@inertiajs/react';

type IndexProps = {
  message: string;
  email: string;
};

export default function Index({  message, email }: IndexProps) {
  return (
    <>
      <Head>
        <title>Hello</title>
      </Head>
      <div>
        <h1>{message}</h1>
        <p>This page is powered by <a href="https://github.com/kohkimakimoto/inertia-echo">inertia-echo</a></p>
        <p>Your email is: <strong>{email}</strong></p>
        <p>Click <Link href="/about">here</Link> to go to the About page.</p>
        <p>
          <Link href="/logout">Logout</Link>
        </p>
      </div>
    </>
  );
}
