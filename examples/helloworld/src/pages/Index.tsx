import React from 'react';
import { Head, Link } from '@inertiajs/inertia-react';

type IndexProps = {
  message: string;
};

export default function Index({ message }: IndexProps) {
  return (
    <>
      <Head>
        <title>{message}</title>
      </Head>
      <div>
        {message}
        <div>
          <Link href="/about">About us</Link>
        </div>
      </div>
    </>
  )
}
