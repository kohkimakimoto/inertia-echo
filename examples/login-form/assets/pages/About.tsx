import React from 'react';
import { Head, Link, Deferred } from '@inertiajs/react';

type AboutProps = {
  title: string;
  deferredMessage: string;
};

export default function Index({ title, deferredMessage }: AboutProps) {
    return (
      <>
        <Head>
          <title>{ title }</title>
        </Head>
        <div>
          <h1>About Page</h1>
          <p>The inertia-echo is a Go library that combines Inertia.js and Echo, allowing you to build modern single-page applications.</p>
          <p>Back to <Link href="/">Home</Link></p>
        </div>
      </>
    );
}
