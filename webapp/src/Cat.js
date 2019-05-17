import React from 'react';

class Cat extends React.Component {
    render() {
        return (
            <img src={"data:image/png;base64," + this.props.imageData} alt="cat!" />
        )
    }
}

export default Cat;