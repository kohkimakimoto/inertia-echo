import React from 'react';
import { Head } from '@inertiajs/inertia-react';

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
      </div>
    </>
  )
}
