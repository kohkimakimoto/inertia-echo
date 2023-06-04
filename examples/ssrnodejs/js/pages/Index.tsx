import React from 'react';
import { Head } from '@inertiajs/react';

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
      <div className="bg-white flex-col text-center pt-7">
        <h1 className="text-4xl mt-3 mb-3">{ message }</h1>
        <p>This page is powered by <a className="text-blue-500" href="https://github.com/kohkimakimoto/inertia-echo">inertia-echo</a></p>
      </div>
    </>
  );
}
