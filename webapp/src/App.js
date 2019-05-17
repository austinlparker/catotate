import React from 'react';
import logo from './logo.svg';
import './App.css';

import Cat from './Cat'
import CatContainer from './CatContainer';

function App() {
  return (
    <div className="App">
      <header className="App-header">
        <h1>catotate!</h1>
        <CatContainer />
      </header>
    </div>
  );
}

export default App;
