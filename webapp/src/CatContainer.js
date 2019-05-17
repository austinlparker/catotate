import React from 'react';
import Cat from './Cat';

const API = 'http://localhost:3001/getHelloWorldCat';

class CatContainer extends React.Component {
    constructor(props) {
        super(props);
        this.state = {
            val: '',
            img: ''
        };
    
    this.handleChange = this.handleChange.bind(this);
    this.handleSubmit = this.handleSubmit.bind(this);
    }

    handleChange(event) {
        this.setState({val: event.target.value });
    }

    handleSubmit(event) {
        var uri
        if (this.state.val.length > 0) {
            uri = `${API}?annotation=${this.state.val}`
        } else {
            uri = API
        }
        fetch(uri)
            .then(res => res.text())
            .then(data => this.setState({img: data, val: ''}))
        event.preventDefault()
    }

    render() {
        return (
            <div>
                <form onSubmit={this.handleSubmit}>
                    <label>Annotation:
                        <input type="text" value={this.state.val} onChange={this.handleChange} />
                    </label>
                    <input type="submit" value="submit" />
                </form>
                {(this.state.img.length > 0) ? <Cat imageData={this.state.img} /> : null}
            </div>
        );
    }
}

export default CatContainer;