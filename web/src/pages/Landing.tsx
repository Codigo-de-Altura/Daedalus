import Hero from "../components/landing/Hero";
import Adapters from "../components/landing/Adapters";
import HowItWorks from "../components/landing/HowItWorks";
import Features from "../components/landing/Features";
import Compile from "../components/landing/Compile";
import Comparison from "../components/landing/Comparison";
import CTA from "../components/landing/CTA";

export default function Landing() {
  return (
    <>
      <Hero />
      <Adapters />
      <HowItWorks />
      <Features />
      <Compile />
      <Comparison />
      <CTA />
    </>
  );
}
