import React from 'react';
import { Head, Link } from '@inertiajs/react';

type IndexProps = {
  title: string;
  message: string;
};

export default function Index({ title, message }: IndexProps) {
  return (
    <>
      <Head>
        <title>{ title }</title>
      </Head>
      <div>
        <h1>{ message }</h1>
        <p>This page is SSR'd in-process by <a href="https://github.com/liustack/slowjs">slowjs</a> via <a href="https://github.com/kohkimakimoto/inertia-echo">inertia-echo</a>.</p>
        <p>Click <Link href="/about">here</Link> to go to the About page.</p>
      </div>
    </>
  );
}
